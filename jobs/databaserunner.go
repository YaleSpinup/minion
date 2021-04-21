package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
)

type DatabaseRunner struct {
	Endpoint         string
	EndpointTemplate string
	Token            string
}

// NewDatabaseRunner creates and configures a new database runner.  An endpoint or endpoint template is required
// the endpoint and endpoint template are not currently validated but this can/should be done in the future.  If a
// a token is passed, it will be configured but is not required.
func NewDatabaseRunner(config map[string]interface{}) (*DatabaseRunner, error) {
	log.Error("creating new database job runner")

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

	return &DatabaseRunner{
		Endpoint:         endpoint,
		EndpointTemplate: endpointTemplate,
		Token:            token,
	}, nil
}

// Run executes the DatabaseRunner.  The instance_id and database_action are required.  Allowable actions
// are 'start' and 'stop'.  If an endpoint is configured on the runner, it will be used, otherwise
// we assume there is an endpointTemplate and try to execute it.  Database actions are currently only
// executed with the PUT method and a body containing {"state": "action"}.
func (r *DatabaseRunner) Run(ctx context.Context, account string, parameters interface{}) (string, error) {
	if account == "" {
		return "", errors.New("account is required")
	}

	log.Infof("running database runner %+v in account %s,  with parameters %+v", r, account, parameters)

	params, ok := parameters.(map[string]string)
	if !ok {
		return "", NewRunnerError(ErrMissingDetails, "wrong type", errors.New("parameters list is not a map[string]string"))
	}

	instanceID, ok := params["instance_id"]
	if !ok {
		return "", NewRunnerError(ErrMissingDetails, "missing instance_id", nil)
	}

	action, ok := params["database_action"]
	if !ok {
		return "", NewRunnerError(ErrMissingDetails, "missing database_action", nil)
	}

	endpoint := r.Endpoint
	if endpoint == "" {
		log.Debugf("endpoint is empty, attempting to use endpoint template '%s'", r.EndpointTemplate)

		input := struct {
			Account    string
			InstanceID string
		}{account, instanceID}

		tmpl, err := template.New("endpoint").Parse(r.EndpointTemplate)
		if err != nil {
			return "", NewRunnerError(ErrPreExecFailure, "template parsing failed", err)
		}

		var out bytes.Buffer
		if err := tmpl.Execute(&out, &input); err != nil {
			return "", NewRunnerError(ErrPreExecFailure, "template execution failed", err)
		}

		endpoint = out.String()
		log.Debugf("parsed endpoint template: %s", endpoint)
	}

	switch action {
	case "stop", "start":
		j, err := json.Marshal(map[string]string{"state": action})
		if err != nil {
			return "", err
		}

		client := &http.Client{
			Timeout: time.Second * 30,
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(j))
		if err != nil {
			return "", NewRunnerError(ErrPreExecFailure, "building http request failed", err)
		}

		if r.Token != "" {
			req.Header.Set("X-Auth-Token", r.Token)
		}
		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			return "", NewRunnerError(ErrExecFailure, "http request failed", err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", NewRunnerError(ErrPostExecFailure, "reading response body failed", err)
		}

		log.Debugf("got response %s(%d) for endpoint %s: %s", res.Status, res.StatusCode, endpoint, body)

		if res.StatusCode >= 300 {
			return "", NewRunnerError(ErrExecFailure, "unexpected http response", errors.New("unexpected response from databaseRunner api: "+res.Status))
		}

		return string(body), nil
	default:
		return "", fmt.Errorf("unexpected action '%s' for database %s", action, instanceID)
	}
}
