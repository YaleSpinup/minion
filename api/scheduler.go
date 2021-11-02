package api

import (
	"context"
	"strconv"
	"time"

	"github.com/YaleSpinup/minion/jobs"
	log "github.com/sirupsen/logrus"
)

// start the scheduler loop
func (s *scheduler) start(ctx context.Context) error {
	log.Infof("%s: scheduler starting", s.id)

	go s.loop(ctx)

	log.Infof("%s: scheduler started", s.id)
	return nil
}

// loop runs the scheduler every minute
func (s *scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		log.Debugf("%s starting scheduler loop (%s)", s.id, time.Now().String())
		select {
		case <-ticker.C:
			basis := time.Now().UTC().Truncate(time.Minute)
			go s.run(ctx, basis)
		case <-ctx.Done():
			log.Debug("shutting down loader timer")
			ticker.Stop()
			return
		}
	}
}

// run does the scheduling of jobs.  first we aquire a central lock, then determines the minute we are in
// and looks for any enabled job in the cache which should be scheduled now.  if any are found, they are enqueued.
func (s *scheduler) run(ctx context.Context, now time.Time) {
	defer timeTrack("scheduler.run()", time.Now())

	log.Debugf("%s acquiring lock", s.id)
	if err := s.locker.Lock(strconv.FormatInt(now.Unix(), 10), s.id); err != nil {
		log.Warnf("%s failed to aquire lock, moving on...", s.id)
		return
	}
	log.Debugf("%s acquired lock", s.id)

	basis := now.Add(time.Duration(-1) * time.Minute).UTC().Truncate(time.Minute)

	log.Infof("%s running jobs scheduler %s with basis time %s", s.id, now.String(), basis.String())

	s.jobsCache.Mux.Lock()
	defer s.jobsCache.Mux.Unlock()

	for id, job := range s.jobsCache.Cache {
		log.Debugf("processing job %s schedule %s", id, job.ScheduleExpression)

		if !job.Enabled {
			log.Debugf("job %s is disabled", id)
			continue
		}

		next, err := job.NextRun(basis)
		if err != nil {
			log.Errorf("failed to get next run for job id '%s' with expression '%s': %s", id, job.ScheduleExpression, err)
			continue
		}

		log.Debugf("%s next execution is %s", id, next.String())

		if next.Equal(now) {
			log.Infof("%s enqueing job %s", s.id, id)
			if err := s.jobQueue.Enqueue(&jobs.QueuedJob{ID: id}); err != nil {
				log.Errorf("failed enqueing job %s: %s", id, err)
			}
		}
	}

	log.Infof("%s done scheduling jobs", s.id)
}
