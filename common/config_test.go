package common

import (
	"bytes"
	"reflect"
	"testing"
)

var testConfig = []byte(
	`{ 
		"accounts": {
		  "myaccount": {
			"runners": ["dummyRunner", "instanceRunner"]
		  }
		},
		"jobsRepository": {
			"type": "s3",
			"RefreshInterval": "60m",
			"config": {
				"region": "us-east-1",
				"akid": "keykeykeykeykeykeykey",
				"secret": "secretsecretsecretsecretsecret",
				"bucket": "myjobsrepository",
				"prefix": "jobs"
			}
		},
		"jobRunners": {
			"dummyRunner": {
				"type": "dummy",
				"config": {
					"template": "Hello, {{.Account}}!"
				}
			},
			"instanceRunner": {
				"type": "instance",
				"config": {
					"endpoint": "http://127.0.0.1:8080/v1/ec2/power",
					"token": "yyyyyyyy"
				}
			}
		},
		"lockProvider": {
			"type": "redis",
			"ttl": "5m",
			"config": {
				"database": 2,
				"host": "127.0.0.1",
				"port": 6379
			}
		},
		"listenAddress": ":8000",
		"token": "SEKRET",
		"logLevel": "info",
		"org": "test"
	  }`)

var brokenConfig = []byte(`{ "foobar": { "baz": "biz" }`)

func TestReadConfig(t *testing.T) {
	expectedConfig := Config{
		Accounts: map[string]Account{
			"myaccount": Account{
				Runners: []string{"dummyRunner", "instanceRunner"},
			},
		},
		JobsRepository: JobsRepository{
			Type:            "s3",
			RefreshInterval: "60m",
			Config: map[string]interface{}{
				"akid":   "keykeykeykeykeykeykey",
				"secret": "secretsecretsecretsecretsecret",
				"bucket": "myjobsrepository",
				"prefix": "jobs",
				"region": "us-east-1",
			},
		},
		JobRunners: map[string]JobRunner{
			"dummyRunner": JobRunner{
				Type: "dummy",
				Config: map[string]interface{}{
					"template": "Hello, {{.Account}}!",
				},
			},
			"instanceRunner": JobRunner{
				Type: "instance",
				Config: map[string]interface{}{
					"endpoint": "http://127.0.0.1:8080/v1/ec2/power",
					"token":    "yyyyyyyy",
				},
			},
		},
		LockProvider: LockProvider{
			Type: "redis",
			TTL:  "5m",
			Config: map[string]interface{}{
				"database": float64(2),
				"host":     "127.0.0.1",
				"port":     float64(6379),
			},
		},
		ListenAddress: ":8000",
		Token:         "SEKRET",
		LogLevel:      "info",
		Org:           "test",
	}

	actualConfig, err := ReadConfig(bytes.NewReader(testConfig))
	if err != nil {
		t.Error("Failed to read config", err)
	}

	if !reflect.DeepEqual(actualConfig, expectedConfig) {
		t.Errorf("Expected config to be %+v\n got %+v", expectedConfig, actualConfig)
	}

	_, err = ReadConfig(bytes.NewReader(brokenConfig))
	if err == nil {
		t.Error("expected error reading config, got nil")
	}
}
