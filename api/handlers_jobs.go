package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/minion/apierror"
	"github.com/YaleSpinup/minion/jobs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// JobCreateHandler creates a new "job" in the repository
func (s *server) JobsCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("creating job for account %s", account)

	input := jobs.Job{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create job input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	log.Debugf("decoded request body into job input %+v", input)

	out, err := s.jobsRepository.Create(r.Context(), account, &input)
	if err != nil {
		handleError(w, err)
		return
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

func (s *server) JobsListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("listing jobs for account %s from repository", account)

	list, err := s.jobsRepository.List(r.Context(), account)
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

func (s *server) JobsShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	id := vars["id"]

	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("showing job %s for account %s from repository", id, account)

	job, err := s.jobsRepository.Get(r.Context(), account, id)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(&job)
	if err != nil {
		msg := fmt.Sprintf("cannot encode job into json: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s *server) JobsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	id := vars["id"]

	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("updating job '%s' for account %s from repository", id, account)

	input := jobs.Job{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create job input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}
	input.ID = id

	log.Debugf("decoded request body into job input %+v", input)

	out, err := s.jobsRepository.Update(r.Context(), account, id, &input)
	if err != nil {
		handleError(w, err)
		return
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

func (s *server) JobsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	id := vars["id"]

	if _, ok := s.accounts[account]; !ok {
		msg := fmt.Sprintf("account not found: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("deleting job '%s' for account %s fromt repository", id, account)

	err := s.jobsRepository.Delete(r.Context(), account, id)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OK"))
}
