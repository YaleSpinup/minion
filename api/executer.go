package api

import (
	"context"
	"time"

	"github.com/YaleSpinup/minion/jobs"
	log "github.com/sirupsen/logrus"
)

func (e *executer) start(ctx context.Context, interval time.Duration) {
	log.Infof("%s: executer starting", e.id)
	go e.loop(ctx, interval)
	log.Infof("%s: executer started", e.id)
}

func (e *executer) loop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		log.Debugf("%s: starting executer loop", e.id)
		select {
		case <-ticker.C:
			q := jobs.QueuedJob{}
			if err := e.jobQueue.Fetch(&q); err != nil {
				qErr, ok := err.(jobs.QueueError)
				if ok && qErr.Code == jobs.ErrQueueIsEmpty {
					log.Debugf("%s: no jobs", e.id)
				} else {
					log.Errorf("%s: error fetching jobs from the queue: %s", e.id, err)
				}
				continue
			}

			if q.ID != "" {
				log.Debugf("%s: about to execute queued job %+v", e.id, q)

				e.jobsCache.Mux.Lock()
				job, ok := e.jobsCache.Cache[q.ID]
				e.jobsCache.Mux.Unlock()

				if !ok {
					log.Warnf("%s: job %s not found in the job cache", e.id, q.ID)
					continue
				}

				// if the job from the repository has a runner configured
				runner, ok := job.Details["runner"]
				if !ok {
					log.Warnf("%s: runner not found in the job details for %s", e.id, job.ID)
					continue
				}

				log.Debugf("%s: found requested runner '%s' in job details", e.id, runner)

				// look for that runner in the list of available runners
				jr, ok := e.jobRunners[runner]
				if !ok {
					log.Warnf("%s: jobRunner not defined for requested runner '%s'", e.id, runner)
					continue
				}

				log.Debugf("%s: jobRunner defined for requested runner '%s': %+v", e.id, runner, jr)

				go e.run(ctx, jr, job)
			}
		case <-ctx.Done():
			log.Debugf("%s: shutting down executer ticker", e.id)
			ticker.Stop()
			return
		}
	}
}

func (e *executer) run(ctx context.Context, runner jobs.Runner, j *jobs.Job) {
	// defer finalizing the job until we return (success or failure)
	defer func() {
		if err := e.jobQueue.Finalize(j.ID); err != nil {
			log.Errorf("%s: error finalizing job %s: %s", e.id, j.ID, err)
		}
	}()

	for i := 1; i <= 3; i++ {
		log.Debugf("running (%d) job executer for %+v", i, j)

		// run the configured runner
		out, err := runner.Run(ctx, j.Account, j.Details)
		if err != nil {
			log.Errorf("failed running job (%d tries) %s: %s", i, j.ID, err)

			timer := time.NewTimer(5 * time.Second)
			select {
			case <-ctx.Done():
				log.Warnf("cancelling retrying of job %s", j.ID)
				timer.Stop()
				return
			case <-timer.C:
				log.Infof("retrying job (%d) %s", i, j.ID)
			}
		}

		log.Debugf("got output from running job: %s", out)
		return
	}
}
