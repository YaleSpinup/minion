package api

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/YaleSpinup/minion/common"
	"github.com/YaleSpinup/minion/jobs"
	"github.com/YaleSpinup/minion/namesgenerator"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	report "github.com/YaleSpinup/eventreporter"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var publicURLs = map[string]string{
	"/v1/minion/ping":    "public",
	"/v1/minion/version": "public",
	"/v1/minion/metrics": "public",
}

// apiVersion is the API version
type apiVersion struct {
	// The version of the API
	Version string `json:"version"`
	// The git hash of the API
	GitHash string `json:"githash"`
	// The build timestamp of the API
	BuildStamp string `json:"buildstamp"`
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
	logger         *logger
	router         *mux.Router
	version        *apiVersion
}

// loader is responsible for loading the jobs from durable storage into a local cache.
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
	logger     *logger
}

var (
	// Org will carry throughout the api and get tagged on resources
	Org string

	EventReporters []report.Reporter
)

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	id := namesgenerator.GetRandomName(0)
	log.Infof("starting api server with id '%s'", id)

	if config.Org == "" {
		return errors.New("'org' cannot be empty in the configuration")
	}
	Org = config.Org

	// TODO: replace this with something else, this is no good
	jobsCache := &jobsCache{
		Cache: make(map[string]*jobs.Job),
	}

	if err := configureEventReporters(config.EventReporters); err != nil {
		return err
	}

	reportEvent(fmt.Sprintf("Starting minion (id: %s, org: %s)", id, Org), report.INFO)

	// setup server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server{
		accounts:   make(map[string]common.Account),
		jobRunners: make(map[string]jobs.Runner),
		logger:     newLogger(Org, config.LogProvider),
		router:     mux.NewRouter(),
	}

	s.version = &apiVersion{
		Version:    config.Version.Version,
		GitHash:    config.Version.GitHash,
		BuildStamp: config.Version.BuildStamp,
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
		logger:     newLogger(Org, config.LogProvider),
	}

	d := scheduler{
		id:        id,
		jobsCache: jobsCache,
	}

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

	// load jobs from durable storage into the local cache
	err = l.start(ctx)
	if err != nil {
		return err
	}

	// pop and execute jobs from the queue
	e.start(ctx, time.Second)

	// start the job scheduler
	if err := d.start(ctx); err != nil {
		return err
	}

	// TODO build up and start requeuer that
	// * checks the backup queue for jobs older than XX minutes and requeues them (assuming the runner died)

	// load routes
	s.routes()

	if config.ListenAddress == "" {
		config.ListenAddress = ":8080"
	}

	handler := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, TokenMiddleware([]byte(config.Token), publicURLs, s.router)))
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
		case "service":
			r, err := jobs.NewServiceRunner(c.Config)
			if err != nil {
				return nil, err
			}
			jobRunners[name] = r

			log.Infof("configured new service runner %s", name)
		case "database":
			r, err := jobs.NewDatabaseRunner(c.Config)
			if err != nil {
				return nil, err
			}
			jobRunners[name] = r

			log.Infof("configured new database runner %s", name)
		default:
			return nil, errors.New("failed to determine jobs runner type, or type not supported: " + c.Type)
		}
	}

	return jobRunners, nil
}

// configureEventReporters configures the global event reporters
func configureEventReporters(configs map[string]common.EventReporterConfig) error {
	for name, config := range configs {
		r, err := report.New(name, config)
		if err != nil {
			return err
		}

		EventReporters = append(EventReporters, r)
	}

	return nil
}

// reportEvent loops over all of the configured event reporters and sends the event to those reporters
func reportEvent(msg string, level report.Level) {
	e := report.Event{
		Message: msg,
		Level:   level,
	}

	for _, r := range EventReporters {
		err := r.Report(e)
		if err != nil {
			log.Errorf("Failed to report event (%s) %s", msg, err.Error())
		}
	}
}
