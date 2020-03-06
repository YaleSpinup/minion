package api

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/YaleSpinup/minion/common"
	"github.com/YaleSpinup/minion/jobs"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type jobRunner struct{}

type refreshRunner struct {
}

type server struct {
	accounts       map[string]common.Account
	context        context.Context
	jobsRepository jobs.Repository
	jobRunners     map[string]jobs.Runner
	router         *mux.Router
	version        common.Version
}

// Org will carry throughout the api and get tagged on resources
var Org string

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	// setup server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server{
		accounts:   make(map[string]common.Account),
		jobRunners: make(map[string]jobs.Runner),
		router:     mux.NewRouter(),
		version:    config.Version,
		context:    ctx,
	}

	if config.Org == "" {
		return errors.New("'org' cannot be empty in the configuration")
	}
	Org = config.Org

	for name, c := range config.Accounts {
		log.Debugf("configuring account %s with %+v", name, c)
		s.accounts[name] = c
	}

	for name, c := range config.JobRunners {
		log.Debugf("configuring job runner %s with %+v", name, c)

		switch c.Type {
		case "dummy":
			r, err := jobs.NewDummyRunner(c.Config)
			if err != nil {
				return err
			}
			s.jobRunners[name] = r

			log.Infof("configured new dummy runner %s", name)
		case "instance":
			r, err := jobs.NewInstanceRunner(c.Config)
			if err != nil {
				return err
			}
			s.jobRunners[name] = r

			log.Infof("configured new instance runner %s", name)
		default:
			return errors.New("failed to determine jobs runner type, or type not supported: " + c.Type)
		}
	}

	repo := config.JobsRepository
	log.Debugf("Creating new JobsRepository of type %s with configuration %+v (org: %s)", repo.Type, repo.Config, Org)

	switch repo.Type {
	case "s3":
		jr, err := jobs.NewDefaultRepository(repo.Config)
		if err != nil {
			return err
		}
		jr.Prefix = jr.Prefix + "/" + Org
		s.jobsRepository = jr
	default:
		return errors.New("failed to determine jobs repository type, or type not supported: " + repo.Type)
	}

	// start job refresher
	// err := refreshJobs()

	publicURLs := map[string]string{
		"/v1/minion/ping":    "public",
		"/v1/minion/version": "public",
		"/v1/minion/metrics": "public",
	}

	// load routes
	s.routes()

	if config.ListenAddress == "" {
		config.ListenAddress = ":8080"
	}
	handler := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, TokenMiddleware(config.Token, publicURLs, s.router)))
	srv := &http.Server{
		Handler:      handler,
		Addr:         config.ListenAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof("Starting listener on %s", config.ListenAddress)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

// LogWriter is an http.ResponseWriter
type LogWriter struct {
	http.ResponseWriter
}

// Write log message if http response writer returns an error
func (w LogWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if err != nil {
		log.Errorf("Write failed: %v", err)
	}
	return
}

// rollBack executes functions from a stack of rollback functions
func rollBack(t *[]func() error) {
	if t == nil {
		return
	}

	tasks := *t
	log.Errorf("executing rollback of %d tasks", len(tasks))
	for i := len(tasks) - 1; i >= 0; i-- {
		f := tasks[i]
		if funcerr := f(); funcerr != nil {
			log.Errorf("rollback task error: %s, continuing rollback", funcerr)
		}
	}
}

type stop struct {
	error
}

// retry is stolen from https://upgear.io/blog/simple-golang-retry-function/
func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2

			time.Sleep(sleep)
			return retry(attempts, 2*sleep, f)
		}
		return err
	}

	return nil
}
