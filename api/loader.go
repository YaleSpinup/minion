package api

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

func (l *loader) start(ctx context.Context) error {
	log.Infof("%s: loader starting", l.id)

	// run the load for the first time which also blocks return until the cache is fresh
	if err := l.run(ctx); err != nil {
		return nil
	}

	go l.loop(ctx)

	log.Infof("%s: loader started", l.id)
	return nil
}

func (l *loader) loop(ctx context.Context) {
	ticker := time.NewTicker(l.refreshInterval)
	for {
		log.Debug("starting loader loop")

		select {
		case <-ticker.C:
			err := l.run(ctx)
			if err != nil {
				log.Errorf("error executing job refresh: %s", err)
			}
		case <-ctx.Done():
			log.Debug("shutting down loader timer")
			ticker.Stop()
			return
		}

		log.Debug("ending loader loop")
	}
}

func (l *loader) run(ctx context.Context) error {
	log.Infof("%s running jobs loader", l.id)

	l.jobsCache.Mux.Lock()
	defer l.jobsCache.Mux.Unlock()
	for name := range l.accounts {
		jobs, err := l.jobsRepository.List(ctx, name)
		if err != nil {
			return err
		}

		log.Debugf("list of jobs: %+v", jobs)

		for _, j := range jobs {
			job, err := l.jobsRepository.Get(ctx, name, j)
			if err != nil {
				log.Errorf("error getting details about job '%s': %s", j, err)
			}

			log.Debugf("got job details: %+v", job)
			l.jobsCache.Cache[j] = job
		}
	}

	log.Infof("%s done loading jobs", l.id)

	return nil
}
