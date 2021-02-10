package common

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config is representation of the configuration data
type Config struct {
	Accounts       map[string]Account
	JobsRepository JobsRepository
	JobRunners     map[string]JobRunner
	ListenAddress  string
	LockProvider   LockProvider
	LogProvider    LogProvider
	Token          string
	LogLevel       string
	QueueProvider  QueueProvider
	Version        Version
	Org            string
}

// Account is the configuration for an individual account
type Account struct {
	Runners []string
}

type JobRunner struct {
	Type   string
	Config map[string]interface{}
}

type JobsRepository struct {
	Type            string
	RefreshInterval string
	Config          map[string]interface{}
}

type LockProvider struct {
	Type   string
	TTL    string
	Config map[string]interface{}
}

type LogProvider struct {
	Region string
	Akid   string
	Secret string
}

type QueueProvider struct {
	Type   string
	Config map[string]interface{}
}

// Version carries around the API version information
type Version struct {
	Version    string
	BuildStamp string
	GitHash    string
}

// ReadConfig decodes the configuration from an io Reader
func ReadConfig(r io.Reader) (Config, error) {
	var c Config
	log.Infoln("Reading configuration")
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return c, errors.Wrap(err, "unable to decode JSON message")
	}
	return c, nil
}
