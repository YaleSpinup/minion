package api

import (
	"context"
	"strconv"
	"time"

	"github.com/YaleSpinup/minion/jobs"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

func (s *scheduler) start(ctx context.Context) error {
	log.Infof("%s: scheduler starting", s.id)

	go s.loop(ctx)

	log.Infof("%s: scheduler started", s.id)
	return nil
}

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

func (s *scheduler) run(ctx context.Context, now time.Time) {
	log.Debugf("%s aquiring lock", s.id)
	if err := s.locker.Lock(strconv.FormatInt(now.Unix(), 10), s.id); err != nil {
		log.Warnf("%s failed to aquire lock, moving on...", s.id)
		return
	}
	log.Debugf("%s aquired lock", s.id)

	basis := now.Add(time.Duration(-1) * time.Minute).UTC().Truncate(time.Minute)

	log.Infof("%s running jobs scheduler %s with basis time %s", s.id, now.String(), basis.String())

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	s.jobsCache.Mux.Lock()
	defer s.jobsCache.Mux.Unlock()

	for id, job := range s.jobsCache.Cache {
		log.Debugf("processing job %s schedule %s", id, job.ScheduleExpression)

		schedule, err := parser.Parse(job.ScheduleExpression)
		if err != nil {
			log.Errorf("%s schedule_expression is not a valid cron expression: '%s': %s", id, job.ScheduleExpression, err)
			continue
		}

		next := schedule.Next(basis)
		log.Debugf("%s next execution is %s", id, next.String())

		if next.Equal(now) {
			log.Infof("%s enqueing job %s", s.id, id)
			s.jobQueue.Enqueue(&jobs.QueuedJob{ID: id})
		}
	}

	log.Infof("%s done scheduling jobs", s.id)
}
