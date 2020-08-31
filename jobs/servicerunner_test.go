package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewServiceRunner(t *testing.T) {
	_, err := NewServiceRunner(map[string]interface{}{})
	if err == nil {
		t.Error("expected error from empty configuration, got nil")
	}

	config := map[string]interface{}{
		"foo": "bar",
	}
	_, err = NewServiceRunner(config)
	if err == nil {
		t.Error("expected error from missing endpoint and endpointTemplate, got nil")
	}

	config["endpointTemplate"] = []string{"biz", "boz", "baz"}
	_, err = NewServiceRunner(config)
	if err == nil {
		t.Error("expected error from endpointTemplate of wrong type, got nil")
	}

	config["endpoint"] = []string{"biz", "boz", "baz"}
	_, err = NewServiceRunner(config)
	if err == nil {
		t.Error("expected error from endpoint of wrong type, got nil")
	}

	expectedEndpointTemplate := "{{.Account}}"
	config["endpointTemplate"] = expectedEndpointTemplate
	_, err = NewServiceRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set endpointTemplate, got %s", err)
	}

	expectedEndpoint := "some endpoint"
	config["endpoint"] = expectedEndpoint
	_, err = NewServiceRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set endpoint, got %s", err)
	}

	expectedToken := "my-awesome-token"
	config["token"] = expectedToken

	out, err := NewServiceRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set token, got %s", err)
	}

	if d := reflect.TypeOf(out).String(); d != "*jobs.ServiceRunner" {
		t.Errorf("expected type to be '*jobs.ServiceRunner', got '%s'", d)
	}

	if out.Endpoint != expectedEndpoint {
		t.Errorf("expected endpoint to be '%s', got '%s'", expectedEndpoint, out.Endpoint)
	}

	if out.EndpointTemplate != expectedEndpointTemplate {
		t.Errorf("expected endpoint token to be '%s', got '%s'", expectedEndpointTemplate, out.EndpointTemplate)
	}

	if out.Token != expectedToken {
		t.Errorf("expected token to be '%s', got '%s'", expectedToken, out.Token)
	}
}

func TestServiceRunnerRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, "unexpected method "+r.Method)
			return
		}

		tok := r.Header.Get("X-Auth-Token")
		t.Logf("got token header with request: %s", tok)

		if err := bcrypt.CompareHashAndPassword([]byte(tok), []byte("my-awesome-token")); err != nil {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "bad token")
			return
		}

		if r.URL.Path != "/v1/ecs/myaccount/clusters/fooclu/services/foosvc" {
			msg := fmt.Sprintf("bad path %s", r.URL.Path)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		inputPayload := struct {
			Service map[string]int
		}{}
		err := json.NewDecoder(r.Body).Decode(&inputPayload)
		if err != nil {
			http.Error(w, "cannot decode body into input", http.StatusBadRequest)
			return
		}

		if inputPayload.Service == nil {
			http.Error(w, "bad input payload", http.StatusBadRequest)
			return
		}

		_, ok := inputPayload.Service["DesiredCount"]
		if !ok {
			http.Error(w, "bad input payload, missing DesiredCount", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	serviceRunner, err := NewServiceRunner(map[string]interface{}{
		"token":         "my-awesome-token",
		"endpoint":      ts.URL,
		"encrypt_token": true,
	})
	if err != nil {
		t.Errorf("expected nil error from set token, got %s", err)
	}

	if _, err = serviceRunner.Run(context.TODO(), "", "foo"); err == nil {
		t.Error("expected 'account is required' error for empty account, got nil")
	} else if err.Error() != "account is required" {
		t.Error("expected 'account is required' error for empty account, got nil")
	}

	if _, err = serviceRunner.Run(context.TODO(), "myaccount", map[string]string{}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected ErrMissingDetails with message 'missing service_action', got %+v", e)
			}

			if e.Message != "missing service_action" {
				t.Errorf("expected ErrMissingDetails with message 'missing service_action', got %+v", e)
			}
		} else {
			t.Error("expected error for missing service_action to be RunnerError")
		}
	} else {
		t.Error("expected error for missing service_action, got nil")
	}

	if _, err = serviceRunner.Run(context.TODO(), "myaccount", map[string]int{"something": 123}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected MissingDetails: wrong type (parameters list is not a map[string]string)', got %+v", e)
			}

			if e.Message != "wrong type parameters list is not a map[string]string" {
				t.Errorf("expected ErrMissingDetails with message 'wrong type (parameters list is not a map[string]string)', got %+v", e)
			}
		} else {
			t.Error("expected error for wrong parameter list type to be RunnerError")
		}
	} else {
		t.Error("expected error for wrong parameter list type, got nil")
	}

	if _, err = serviceRunner.Run(context.TODO(), "myaccount", map[string]string{"service_action": "hax"}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected MissingDetails: unexpected service action 'hax'', got %+v", e)
			}

			if e.Message != "unexpected service action 'hax'" {
				t.Errorf("expected ErrMissingDetails with message 'unexpected service action 'hax'', got %+v", e)
			}
		} else {
			t.Error("expected error for wrong action to be RunnerError")
		}
	} else {
		t.Error("expected error for wrong action, got nil")
	}

	tmplurl := fmt.Sprintf("%s/v1/ecs/{{.Account}}/clusters/{{.Cluster}}/services/{{.Name}}", ts.URL)
	serviceRunner, err = NewServiceRunner(map[string]interface{}{
		"token":            "my-awesome-token",
		"endpointTemplate": tmplurl,
		"encrypt_token":    true,
	})
	if err != nil {
		t.Errorf("expected nil error from set token, got %s", err)
	}

	out, err := serviceRunner.Run(context.TODO(), "myaccount", map[string]string{
		"service_action":  "scale",
		"service_cluster": "fooclu",
		"service_name":    "foosvc",
		"desired_count":   "10",
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if out != "OK" {
		t.Errorf("expected OK, got %s", out)
	}
}

func TestServiceRunnerScaleInputPrep(t *testing.T) {
	type testServiceRunnerScaleInputPrep struct {
		input      *ServiceRunnerScaleInput
		parameters map[string]string
		expected   *ServiceRunnerScaleInput
		err        error
	}

	tests := []*testServiceRunnerScaleInputPrep{
		{
			input:      nil,
			parameters: nil,
			expected:   nil,
			err:        NewRunnerError(ErrMissingDetails, "parameters cannot be nil", nil),
		},
		{
			input: &ServiceRunnerScaleInput{},
			parameters: map[string]string{
				"service_cluster": "fooclu",
				"desired_count":   "123",
			},
			expected: &ServiceRunnerScaleInput{},
			err:      NewRunnerError(ErrMissingDetails, "missing service_name", nil),
		},
		{
			input: &ServiceRunnerScaleInput{},
			parameters: map[string]string{
				"service_name":  "foosvc",
				"desired_count": "123",
			},
			expected: &ServiceRunnerScaleInput{},
			err:      NewRunnerError(ErrMissingDetails, "missing service_cluster", nil),
		},
		{
			input: &ServiceRunnerScaleInput{},
			parameters: map[string]string{
				"service_cluster": "fooclu",
				"service_name":    "foosvc",
			},
			expected: &ServiceRunnerScaleInput{},
			err:      NewRunnerError(ErrMissingDetails, "missing desired_count", nil),
		},
		{
			input: &ServiceRunnerScaleInput{},
			parameters: map[string]string{
				"service_cluster": "fooclu",
				"service_name":    "foosvc",
				"desired_count":   "NaN",
			},
			expected: &ServiceRunnerScaleInput{},
			err:      NewRunnerError(ErrPreExecFailure, "desired count cannot be converted to integer (strconv.Atoi: parsing \"NaN\": invalid syntax)", nil),
		},
		{
			input: &ServiceRunnerScaleInput{},
			parameters: map[string]string{
				"service_cluster": "fooclu",
				"service_name":    "foosvc",
				"desired_count":   "123",
			},
			expected: &ServiceRunnerScaleInput{
				Cluster:      "fooclu",
				Name:         "foosvc",
				desiredCount: 123,
			},
			err: nil,
		},
	}

	for _, test := range tests {
		t.Logf("testing serviceRunner scale input: '%+v', parameters: '%+v'", test.input, test.parameters)

		err := test.input.prep(test.parameters)
		if err != nil && test.err != nil {
			if err.Error() != test.err.Error() {
				t.Errorf("expected error %s, got error %s", test.err, err)
			}
		} else if err != nil && test.err == nil {
			t.Errorf("expected nil error, got %s", err)
		} else if test.err != nil && err == nil {
			t.Errorf("expected error '%s', got nil", test.err)
		} else {
			if !reflect.DeepEqual(test.input, test.expected) {
				t.Errorf("expected '%+v', got '%+v'", test.expected, test.input)
			}
		}
	}

}

func TestExecEndpointTemplate(t *testing.T) {
	type testExecEndpointTemplate struct {
		tmpl     string
		data     interface{}
		expected string
		err      error
	}

	tests := []*testExecEndpointTemplate{
		{
			tmpl:     "",
			data:     "",
			expected: "",
			err:      NewRunnerError(ErrPreExecFailure, "endpoint template can't be empty", nil),
		},
		{
			tmpl:     "",
			data:     map[string]string{"foo": "bar"},
			expected: "",
			err:      NewRunnerError(ErrPreExecFailure, "endpoint template can't be empty", nil),
		},
		{
			tmpl:     "foobar",
			data:     "",
			expected: "foobar",
			err:      nil,
		},
		{
			tmpl:     "foobar",
			data:     map[string]string{"foo": "bar"},
			expected: "foobar",
			err:      nil,
		},
		{
			tmpl:     "foo {{.foo}}",
			data:     map[string]string{"foo": "bar"},
			expected: "foo bar",
			err:      nil,
		},
	}

	for _, test := range tests {
		t.Logf("testing template: '%s', data: '%+v'", test.tmpl, test.data)

		out, err := execEndpointTemplate(test.tmpl, test.data)
		if err != nil && test.err != nil {
			if err.Error() != test.err.Error() {
				t.Errorf("expected error %s, got error %s", err, test.err)
			}
		} else if err != nil && test.err == nil {
			t.Errorf("expected nil error, got %s", err)
		} else if test.err != nil && err == nil {
			t.Errorf("expected error '%s', got nil", test.err)
		} else {
			if out != test.expected {
				t.Errorf("expected %s, got %s", test.expected, out)
			}
		}
	}
}
