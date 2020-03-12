package api

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/YaleSpinup/minion/common"
	"github.com/YaleSpinup/minion/jobs"
	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var publicURLs = map[string]string{
	"/v1/minion/ping":    "public",
	"/v1/minion/version": "public",
	"/v1/minion/metrics": "public",
}

// jobsCache is a map and a mux
type jobsCache struct {
	Cache map[string]*jobs.Job
	Mux   sync.Mutex
}

// server is responsible for all things api.  it caries the configuration
// and dependencies that are necessary in the http handlers.
type server struct {
	accounts       map[string]common.Account
	jobQueue       jobs.Queuer
	jobsRepository jobs.Repository
	jobRunners     map[string]jobs.Runner
	router         *mux.Router
	version        common.Version
}

// loader is responisble for loading the jobs from durable storage into a local cache.
type loader struct {
	accounts        map[string]common.Account
	id              string
	jobsCache       *jobsCache
	jobsRepository  jobs.Repository
	refreshInterval time.Duration
}

// scheduler searches through the locally cached jobs and adds them to the queue
type scheduler struct {
	id        string
	jobsCache *jobsCache
	locker    jobs.Locker
	jobQueue  jobs.Queuer
}

// executer pulls jobs off of the queue and runs then
type executer struct {
	accounts   map[string]common.Account
	id         string
	jobsCache  *jobsCache
	jobQueue   jobs.Queuer
	jobRunners map[string]jobs.Runner
}

// Org will carry throughout the api and get tagged on resources
var Org string

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	id := uuid.New().String()
	log.Infof("starting api server with id '%s'", id)

	// TODO: replace this with something else, this is no good
	jobsCache := &jobsCache{
		Cache: make(map[string]*jobs.Job),
	}

	// setup server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server{
		accounts:   make(map[string]common.Account),
		jobRunners: make(map[string]jobs.Runner),
		router:     mux.NewRouter(),
		version:    config.Version,
	}

	l := loader{
		accounts:  make(map[string]common.Account),
		id:        id,
		jobsCache: jobsCache,
	}

	e := executer{
		accounts:   make(map[string]common.Account),
		id:         id,
		jobRunners: make(map[string]jobs.Runner),
		jobsCache:  jobsCache,
	}

	d := scheduler{
		id:        id,
		jobsCache: jobsCache,
	}

	if config.Org == "" {
		return errors.New("'org' cannot be empty in the configuration")
	}
	Org = config.Org

	for name, c := range config.Accounts {
		log.Debugf("configuring account %s with %+v", name, c)
		s.accounts[name] = c
		l.accounts[name] = c
		e.accounts[name] = c
	}

	jobQueue, err := newJobQueue(Org, config.QueueProvider)
	if err != nil {
		return err
	}
	defer jobQueue.Close()
	s.jobQueue = jobQueue
	e.jobQueue = jobQueue
	d.jobQueue = jobQueue

	jobRunners, err := newJobRunners(Org, config.JobRunners)
	if err != nil {
		return err
	}
	s.jobRunners = jobRunners
	e.jobRunners = jobRunners

	jobsRepository, err := newJobsRepository(Org, config.JobsRepository)
	if err != nil {
		return err
	}
	s.jobsRepository = jobsRepository
	l.jobsRepository = jobsRepository

	refreshInterval, err := time.ParseDuration(config.JobsRepository.RefreshInterval)
	if err != nil {
		return err
	}
	l.refreshInterval = refreshInterval

	// configure the locking mechanism for the scheduler
	locker, err := newLocker(Org, config.LockProvider)
	if err != nil {
		return err
	}
	d.locker = locker

	// load jobs from durable storage into aa local cache
	err = l.start(ctx)
	if err != nil {
		return err
	}

	// pop and execute jobs from the queue
	e.start(ctx, time.Second)

	// start the job scheduler
	d.start(ctx)

	// TODO build up and start requeuer that
	// * checks the backup queue for jobs older than XX minutes and requeues them (assuming the runner died)

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

func newLocker(org string, lp common.LockProvider) (jobs.Locker, error) {
	log.Debugf("configuring locker with %+v", lp)

	var host string
	if hi, ok := lp.Config["host"]; !ok {
		return nil, errors.New("redis host is required")
	} else {
		if h, ok := hi.(string); !ok {
			return nil, errors.New("redis host is required and must be a string")
		} else {
			host = h
		}
	}

	var port string
	if pi, ok := lp.Config["port"]; !ok {
		return nil, errors.New("redis port is required")
	} else {
		log.Debugf("port interface exists %+v", pi)

		if p, ok := pi.(string); ok {
			port = p
		}

		if p, ok := pi.(float64); ok {
			port = strconv.Itoa(int(p))
		}

		if port == "" {
			return nil, errors.New("redis port is required")
		}
	}

	address := host + ":" + port

	var password string
	if pass, ok := lp.Config["password"]; ok {
		if p, ok := pass.(string); ok {
			password = p
		}
	}

	var db int
	if database, ok := lp.Config["database"]; ok {
		if d, ok := database.(string); ok {
			if i, err := strconv.ParseInt(d, 10, 64); err != nil {
				log.Warnf("database '%s' is not parsable as an integer, ignoring", d)
			} else {
				db = int(i)
			}
		} else if d, ok := database.(float64); ok {
			db = int(d)
		} else {
			log.Warnf("database '%v' is not a string or integer, ignoring", database)
		}
	}

	lockerName := "minion-" + org + "-lock"
	locker, err := jobs.NewRedisLocker(lockerName, address, password, db, "2m")
	if err != nil {
		return nil, err
	}
	return locker, nil
}

func newJobQueue(org string, qp common.QueueProvider) (jobs.Queuer, error) {
	log.Debugf("configuring queue with %+v", qp)

	var host string
	if hi, ok := qp.Config["host"]; !ok {
		return nil, errors.New("redis host is required")
	} else {
		if h, ok := hi.(string); !ok {
			return nil, errors.New("redis host is required and must be a string")
		} else {
			host = h
		}
	}

	var port string
	if pi, ok := qp.Config["port"]; !ok {
		return nil, errors.New("redis port is required")
	} else {
		log.Debugf("port interface exists %+v", pi)

		if p, ok := pi.(string); ok {
			port = p
		}

		if p, ok := pi.(float64); ok {
			port = strconv.Itoa(int(p))
		}

		if port == "" {
			return nil, errors.New("redis port is required")
		}
	}

	address := host + ":" + port

	var password string
	if pass, ok := qp.Config["password"]; ok {
		if p, ok := pass.(string); ok {
			password = p
		}
	}

	var db int
	if database, ok := qp.Config["database"]; ok {
		if d, ok := database.(string); ok {
			if i, err := strconv.ParseInt(d, 10, 64); err != nil {
				log.Warnf("database '%s' is not parsable as an integer, ignoring", d)
			} else {
				db = int(i)
			}
		} else if d, ok := database.(float64); ok {
			db = int(d)
		} else {
			log.Warnf("database '%v' is not a string or integer, ignoring", database)
		}
	}

	// setup job queue
	queueName := "minion-" + org + "-queue"
	queue, err := jobs.NewRedisQueuer(queueName, address, password, db, 10)
	if err != nil {
		return nil, err
	}
	return queue, nil
}

func newJobsRepository(org string, repo common.JobsRepository) (jobs.Repository, error) {
	// the jobs repository is the durable storage for jobs
	log.Debugf("Creating new JobsRepository of type %s with configuration %+v (org: %s)", repo.Type, repo.Config, Org)

	switch repo.Type {
	case "s3":
		jr, err := jobs.NewDefaultRepository(repo.Config)
		if err != nil {
			return nil, err
		}
		jr.Prefix = jr.Prefix + "/" + org
		return jr, nil
	}

	return nil, errors.New("failed to determine jobs repository type, or type not supported: " + repo.Type)
}

func newJobRunners(org string, runners map[string]common.JobRunner) (map[string]jobs.Runner, error) {
	jobRunners := make(map[string]jobs.Runner)
	for name, c := range runners {
		log.Debugf("configuring job runner %s with %+v", name, c)

		switch c.Type {
		case "dummy":
			r, err := jobs.NewDummyRunner(c.Config)
			if err != nil {
				return nil, err
			}
			jobRunners[name] = r

			log.Infof("configured new dummy runner %s", name)
		case "instance":
			r, err := jobs.NewInstanceRunner(c.Config)
			if err != nil {
				return nil, err
			}
			jobRunners[name] = r

			log.Infof("configured new instance runner %s", name)
		default:
			return nil, errors.New("failed to determine jobs runner type, or type not supported: " + c.Type)
		}
	}

	return jobRunners, nil
}
