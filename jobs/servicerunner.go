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
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type ServiceRunner struct {
	Endpoint         string
	EndpointTemplate string
	Token            string
	Encrypt          bool
	AuthHeader       string
}

type ServiceRunnerScaleInput struct {
	Account      string
	Cluster      string
	Name         string
	desiredCount int
	endpoint     string
}

// NewServiceRunner creates and configures a new service runner.  An endpoint or endpoint template is required
// the endpoint and endpoint template are not currently validated but this can/should be done in the future.  If a
// a token is passed, it will be configured but is not required.
func NewServiceRunner(config map[string]interface{}) (*ServiceRunner, error) {
	log.Debug("creating new service job runner")

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

	var encrypt bool
	if e, ok := config["encrypt_token"].(bool); ok {
		encrypt = e
	}

	authHeader := "X-Auth-Token"
	if h, ok := config["auth_header"].(string); ok {
		authHeader = h
	}

	return &ServiceRunner{
		Endpoint:         endpoint,
		EndpointTemplate: endpointTemplate,
		Token:            token,
		Encrypt:          encrypt,
		AuthHeader:       authHeader,
	}, nil
}

// Run executes the ServiceRunner.  The service_cluster, service_name and service_action are required.  Allowable
// actions are 'scale'.  If an endpoint is configured on the runner, it will be used, otherwise
// we assume there is an endpointTemplate and try to execute it.
func (r *ServiceRunner) Run(ctx context.Context, account string, parameters interface{}) (string, error) {
	if account == "" {
		return "", errors.New("account is required")
	}

	log.Infof("running service runner %+v in account %s,  with parameters %+v", r, account, parameters)

	params, ok := parameters.(map[string]string)
	if !ok {
		return "", NewRunnerError(ErrMissingDetails, "wrong type parameters list is not a map[string]string", nil)
	}

	action, ok := params["service_action"]
	if !ok {
		return "", NewRunnerError(ErrMissingDetails, "missing service_action", nil)
	}

	switch action {
	case "scale":
		s := &ServiceRunnerScaleInput{
			Account:  account,
			endpoint: r.Endpoint,
		}

		if err := s.prep(params); err != nil {
			return "", NewRunnerError(ErrPreExecFailure, "failed to define input from parameters", err)
		}

		if s.endpoint == "" {
			e, err := execEndpointTemplate(r.EndpointTemplate, s)
			if err != nil {
				return "", NewRunnerError(ErrPreExecFailure, "failed set endpoint", err)
			}
			s.endpoint = e
		}

		inputPayload := struct {
			Service map[string]int
		}{
			map[string]int{
				"DesiredCount": s.desiredCount,
			},
		}

		j, err := json.Marshal(inputPayload)
		if err != nil {
			return "", err
		}

		log.Debugf("scaling %s/%s with input %s", s.Cluster, s.Name, string(j))

		client := &http.Client{
			Timeout: time.Second * 30,
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.endpoint, bytes.NewReader(j))
		if err != nil {
			return "", NewRunnerError(ErrPreExecFailure, "building http request failed", err)
		}

		if r.Token != "" {
			log.Debugf("setting token header %s for %s/%s", r.AuthHeader, s.Cluster, s.Name)
			if r.Encrypt {
				e, err := bcrypt.GenerateFromPassword([]byte(r.Token), 6)
				if err != nil {
					return "", NewRunnerError(ErrExecFailure, "unable to hash token", err)
				}

				log.Debugf("token is encrypted, setting header to %s", string(e))

				req.Header.Set(r.AuthHeader, string(e))
			} else {
				req.Header.Set(r.AuthHeader, r.Token)
			}
		}

		res, err := client.Do(req)
		if err != nil {
			return "", NewRunnerError(ErrExecFailure, "http request failed", err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", NewRunnerError(ErrPostExecFailure, "reading response body failed", err)
		}

		log.Debugf("got response %s(%d) for endpoint %s: %s", res.Status, res.StatusCode, s.endpoint, body)

		if res.StatusCode >= 300 {
			return "", NewRunnerError(ErrExecFailure, "unexpected http response", errors.New("unexpected response from serviceRunner api: "+res.Status))
		}

		return string(body), nil
	default:
		msg := fmt.Sprintf("unexpected service action '%s'", action)
		return "", NewRunnerError(ErrMissingDetails, msg, nil)
	}
}

func (s *ServiceRunnerScaleInput) prep(parameters map[string]string) error {
	if parameters == nil {
		return NewRunnerError(ErrMissingDetails, "parameters cannot be nil", nil)
	}

	log.Debugf("prepping service scale input with params: %+v", parameters)

	serviceCluster, ok := parameters["service_cluster"]
	if !ok {
		return NewRunnerError(ErrMissingDetails, "missing service_cluster", nil)
	}
	s.Cluster = serviceCluster

	serviceName, ok := parameters["service_name"]
	if !ok {
		return NewRunnerError(ErrMissingDetails, "missing service_name", nil)
	}
	s.Name = serviceName

	countString, ok := parameters["desired_count"]
	if !ok {
		return NewRunnerError(ErrMissingDetails, "missing desired_count", nil)
	}

	desiredCount, err := strconv.Atoi(countString)
	if err != nil {
		return NewRunnerError(ErrPreExecFailure, "desired count cannot be converted to integer", err)
	}
	s.desiredCount = desiredCount

	return nil
}

// execEndpointTemplate parses the passed template and then executes it with the given data.
// TODO: use this centrally for other runners
func execEndpointTemplate(tmpl string, data interface{}) (string, error) {
	if tmpl == "" {
		return "", NewRunnerError(ErrPreExecFailure, "endpoint template can't be empty", nil)
	}

	parsedTemplate, err := template.New("endpoint").Parse(tmpl)
	if err != nil {
		return "", NewRunnerError(ErrPreExecFailure, "template parsing failed", err)
	}

	var out bytes.Buffer
	if err := parsedTemplate.Execute(&out, data); err != nil {
		return "", NewRunnerError(ErrPreExecFailure, "template execution failed", err)
	}

	endpoint := out.String()

	log.Debugf("parsed endpoint template: %s", endpoint)

	return endpoint, nil
}
