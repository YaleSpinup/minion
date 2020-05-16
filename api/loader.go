package api

import (
	"context"
	"strings"
	"time"

	"github.com/YaleSpinup/minion/jobs"
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

	cache := make(map[string]*jobs.Job)
	for name := range l.accounts {
		jobs, err := l.jobsRepository.List(ctx, name, "")
		if err != nil {
			return err
		}

		log.Debugf("list of jobs: %+v", jobs)

		for _, j := range jobs {
			var id, group string
			if split := strings.SplitN(j, "/", 2); len(split) == 1 {
				id = split[0]
			} else {
				group = split[0]
				id = split[1]
			}

			job, err := l.jobsRepository.Get(ctx, name, group, id)
			if err != nil {
				log.Errorf("error getting details about job '%s': %s", j, err)
				continue
			}

			if job.Enabled {
				log.Infof("job '%s' is disabled, not caching", id)
				continue
			}

			log.Debugf("caching job id %s with details: %+v", j, job)
			l.jobsCache.Cache[j] = job
			cache[j] = job
		}
	}

	for k := range l.jobsCache.Cache {
		if _, ok := cache[k]; !ok {
			delete(l.jobsCache.Cache, k)
		}
	}

	log.Infof("%s done loading %d jobs", l.id, len(l.jobsCache.Cache))

	return nil
}
