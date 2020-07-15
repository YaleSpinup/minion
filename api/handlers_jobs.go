package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/minion/cloudwatchlogs"
	"github.com/YaleSpinup/minion/jobs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// JobCreateHandler creates a new "job" in the repository
func (s *server) JobsCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]

	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("creating job for account %s, group %s", account, group)

	input := struct {
		Job  *jobs.Job
		Tags []*tag
	}{}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create job input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	if input.Job == nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "job cannot be nil", nil))
		return
	}
	input.Job.Account = account
	input.Job.Group = group

	log.Debugf("decoded request body into job input %+v", input)

	// setup rollback function list and defer execution, note that we depend on the err variable defined above this
	var rollBackTasks []func() error
	defer func() {
		if err != nil {
			log.Errorf("recovering from error creating job: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	job, err := s.jobsRepository.Create(r.Context(), account, group, input.Job)
	if err != nil {
		handleError(w, err)
		return
	}

	// append job cleanup to rollback tasks
	rollBackTasks = append(rollBackTasks, func() error {
		return func() error {
			if err := s.jobsRepository.Delete(r.Context(), account, group, job.ID); err != nil {
				return err
			}
			return nil
		}()
	})

	if err := s.logger.createLog(r.Context(), group, job.ID, int64(90), input.Tags); err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "failed creating job audit log", err))
		return
	}

	out := struct {
		Job  *jobs.Job `json:"job"`
		Tags []*tag    `json:"tags"`
	}{
		Job:  job,
		Tags: input.Tags,
	}

	j, err := json.Marshal(&out)
	if err != nil {
		msg := fmt.Sprintf("cannot encode job output into json: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// JobsListHandler lists the jobs in the repository for an account
func (s *server) JobsListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	group := vars["group"]
	account := vars["account"]
	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("listing jobs for account '%s', group '%s' from repository", account, group)

	list, err := s.jobsRepository.List(r.Context(), account, group)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(&list)
	if err != nil {
		msg := fmt.Sprintf("cannot encode job listing into json: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// JobsShowHandler gets the details about an individual job in the repository
func (s *server) JobsShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]
	id := vars["id"]

	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("showing job %s for account %s from repository", id, account)

	job, err := s.jobsRepository.Get(r.Context(), account, group, id)
	if err != nil {
		handleError(w, err)
		return
	}

	lg, tags, err := s.logger.describeLog(r.Context(), group)
	if err != nil {
		handleError(w, err)
		return
	}

	out := struct {
		Job  *jobs.Job                `json:"job"`
		Tags []*tag                   `json:"tags"`
		Log  *cloudwatchlogs.LogGroup `json:"log"`
	}{
		Job:  job,
		Tags: tags,
		Log:  lg,
	}

	j, err := json.Marshal(&out)
	if err != nil {
		msg := fmt.Sprintf("cannot encode job into json: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// JobsUpdateHandler updates the details of a job
func (s *server) JobsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]
	id := vars["id"]

	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("updating job '%s' for account %s, group %s", id, account, group)

	input := struct {
		Job  *jobs.Job
		Tags []*tag
	}{}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create job input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	if input.Job == nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "job cannot be nil", nil))
		return
	}
	input.Job.ID = id
	input.Job.Account = account
	input.Job.Group = group

	log.Debugf("decoded request body into job input %+v", input)

	// get the job to be sure it exists
	if _, err := s.jobsRepository.Get(r.Context(), account, group, id); err != nil {
		handleError(w, err)
		return
	}

	job, err := s.jobsRepository.Update(r.Context(), account, group, id, input.Job)
	if err != nil {
		handleError(w, err)
		return
	}

	if err := s.logger.updateLog(r.Context(), group, int64(90), input.Tags); err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "failed updating job audit log", err))
		return
	}

	lg, tags, err := s.logger.describeLog(r.Context(), group)
	if err != nil {
		handleError(w, err)
		return
	}

	out := struct {
		Job  *jobs.Job                `json:"job"`
		Tags []*tag                   `json:"tags"`
		Log  *cloudwatchlogs.LogGroup `json:"log"`
	}{
		Job:  job,
		Tags: tags,
		Log:  lg,
	}

	j, err := json.Marshal(&out)
	if err != nil {
		msg := fmt.Sprintf("cannot encode job into json: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(j)
}

// JobsDeleteHandler removes a job from the respository
func (s *server) JobsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]
	id := vars["id"]

	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("deleting job %s/%s/%s from repository", account, group, id)

	err := s.jobsRepository.Delete(r.Context(), account, group, id)
	if err != nil {
		handleError(w, err)
		return
	}

	// TODO archive cloudwatchlog log stream

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OK"))
}

// JobsRunHandler runs a job explicitely.  This will probably be used mainly for testing and may go away.
func (s *server) JobsRunHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	acct := vars["account"]
	group := vars["group"]
	id := vars["id"]

	account, ok := s.accounts[acct]
	if !ok {
		msg := fmt.Sprintf("account not found: %s", acct)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Debugf("queuing job %s/%s/%s", acct, group, id)

	// get the job details from the jobs repostory
	job, err := s.jobsRepository.Get(r.Context(), acct, group, id)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// if the job from the repository has a runner configured
	if runner, ok := job.Details["runner"]; ok {
		log.Debugf("found requested runner '%s' in job details", runner)

		// look for that runner in the list of available runners
		if jobRunner, ok := s.jobRunners[runner]; ok {
			log.Debugf("jobRunner is defined for requested runner '%s': %+v", runner, jobRunner)

			// check if the runner is configured for the account
			allowed := false
			for _, r := range account.Runners {
				if r == runner {
					allowed = true
					break
				}
			}

			if !allowed {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("runner not found for account"))
			}

			if err := s.jobQueue.Enqueue(&jobs.QueuedJob{ID: group + "/" + id}); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("failed queuing job " + err.Error()))
				return
			}

			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("OK"))
			return
		}
		log.Warnf("jobRunner is not defined for requested runner '%s'", runner)
	}

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("runner not found in job"))
}
