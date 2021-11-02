package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type TaskRunner struct {
	Endpoint         string
	EndpointTemplate string
	Token            string
	Encrypt          bool
	AuthHeader       string
}

type TaskRunnerRunInput struct {
	Account  string
	Cluster  string
	Name     string
	count    int
	endpoint string
}

// NewTaskRunner creates and configures a new task runner.  An endpoint or endpoint template is required
// the endpoint and endpoint template are not currently validated but this can/should be done in the future.
// If a token is passed, it will be configured but is not required.
func NewTaskRunner(config map[string]interface{}) (*TaskRunner, error) {
	log.Debug("creating new task job runner")

	var endpoint string
	if v, ok := config["endpoint"].(string); ok {
		endpoint = v
	}

	var endpointTemplate string
	if v, ok := config["endpointTemplate"].(string); ok {
		endpointTemplate = v
	}

	if endpoint == "" && endpointTemplate == "" {
		return nil, errors.New("endpoint or endpoint_template is required")
	} else if endpoint != "" && endpointTemplate != "" {
		log.Warn("both endpoint and endpoint_template are set, only endpoint will be used")
	}

	var token string
	if v, ok := config["token"].(string); ok {
		token = v
	}

	// default to encrypting the token
	encrypt := true
	if e, ok := config["encrypt_token"].(bool); ok {
		encrypt = e
	}

	// default to the X-Auth-Token header
	authHeader := "X-Auth-Token"
	if h, ok := config["auth_header"].(string); ok {
		authHeader = h
	}

	return &TaskRunner{
		Endpoint:         endpoint,
		EndpointTemplate: endpointTemplate,
		Token:            token,
		Encrypt:          encrypt,
		AuthHeader:       authHeader,
	}, nil
}

// Run executes the TaskRunner.  The task_cluster, task_name and task_action are required.  Allowable
// actions are 'run'.  If an endpoint is configured on the runner, it will be used, otherwise
// we assume there is an endpointTemplate and try to execute it.
func (r *TaskRunner) Run(ctx context.Context, account string, parameters interface{}) (string, error) {
	if account == "" {
		return "", errors.New("account is required")
	}

	log.Debugf("initializing task runner %+v in account %s,  with parameters %+v", r, account, parameters)

	params, ok := parameters.(map[string]string)
	if !ok {
		return "", NewRunnerError(ErrMissingDetails, "wrong type parameters list is not a map[string]string", nil)
	}

	action, ok := params["task_action"]
	if !ok {
		return "", NewRunnerError(ErrMissingDetails, "missing task_action", nil)
	}

	switch action {
	case "run":
		i := &TaskRunnerRunInput{
			Account:  account,
			endpoint: r.Endpoint,
		}

		if err := i.prep(params); err != nil {
			return "", NewRunnerError(ErrPreExecFailure, "failed to prep input from parameters", err)
		}

		if i.endpoint == "" {
			e, err := execEndpointTemplate(r.EndpointTemplate, i)
			if err != nil {
				return "", NewRunnerError(ErrPreExecFailure, "failed set endpoint", err)
			}
			i.endpoint = e
		}

		inputPayload := struct {
			Count     int
			StartedBy string
		}{
			Count:     i.count,
			StartedBy: "minion",
		}

		j, err := json.Marshal(inputPayload)
		if err != nil {
			return "", err
		}

		log.Debugf("task runner run %s/%s with input %s", i.Cluster, i.Name, string(j))

		client := &http.Client{
			Timeout: time.Second * 30,
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.endpoint, bytes.NewReader(j))
		if err != nil {
			return "", NewRunnerError(ErrPreExecFailure, "building http request failed", err)
		}

		if r.Token != "" {
			log.Debugf("setting token header %s for %s/%s", r.AuthHeader, i.Cluster, i.Name)
			if r.Encrypt {
				e, err := bcrypt.GenerateFromPassword([]byte(r.Token), 6)
				if err != nil {
					return "", NewRunnerError(ErrExecFailure, "unable to hash token", err)
				}

				log.Debug("token is encrypted")

				req.Header.Set(r.AuthHeader, string(e))
			} else {
				req.Header.Set(r.AuthHeader, r.Token)
			}
		}

		log.Infof("task runner running  %s/%s with count %d", i.Cluster, i.Name, i.count)

		res, err := client.Do(req)
		if err != nil {
			return "", NewRunnerError(ErrExecFailure, "http request failed", err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", NewRunnerError(ErrPostExecFailure, "reading response body failed", err)
		}

		log.Debugf("got response %s(%d) for endpoint %s: %s", res.Status, res.StatusCode, i.endpoint, body)

		if res.StatusCode >= 300 {
			return "", NewRunnerError(ErrExecFailure, "unexpected http response", errors.New("unexpected response from taskRunner api: "+res.Status))
		}

		msg := fmt.Sprintf("successfully submitted run task %s/%s with count %d", i.Cluster, i.Name, i.count)
		return msg, nil
	default:
		msg := fmt.Sprintf("unexpected task action '%s'", action)
		return "", NewRunnerError(ErrMissingDetails, msg, nil)
	}
}

// prep parses the job parameters and sets up the task runner input
func (i *TaskRunnerRunInput) prep(parameters map[string]string) error {
	if parameters == nil {
		return NewRunnerError(ErrMissingDetails, "parameters cannot be nil", nil)
	}

	log.Debugf("prepping service scale input with params: %+v", parameters)

	taskCluster, ok := parameters["task_cluster"]
	if !ok {
		return NewRunnerError(ErrMissingDetails, "missing task_cluster", nil)
	}
	i.Cluster = taskCluster

	taskName, ok := parameters["task_name"]
	if !ok {
		return NewRunnerError(ErrMissingDetails, "missing task_name", nil)
	}
	i.Name = taskName

	countString, ok := parameters["count"]
	if !ok {
		return NewRunnerError(ErrMissingDetails, "missing count", nil)
	}

	count, err := strconv.Atoi(countString)
	if err != nil {
		return NewRunnerError(ErrPreExecFailure, "count cannot be converted to integer", err)
	}

	if count <= 0 {
		return NewRunnerError(ErrPreExecFailure, "count cannot be 0", nil)
	}

	i.count = count

	return nil
}
