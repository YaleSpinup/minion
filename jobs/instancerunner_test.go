package jobs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewInstanceRunner(t *testing.T) {
	_, err := NewInstanceRunner(map[string]interface{}{})
	if err == nil {
		t.Error("expected error from empty configuration, got nil")
	}

	config := map[string]interface{}{
		"foo": "bar",
	}
	_, err = NewInstanceRunner(config)
	if err == nil {
		t.Error("expected error from missing endpoint and endpointTemplate, got nil")
	}

	config["endpointTemplate"] = []string{"biz", "boz", "baz"}
	_, err = NewInstanceRunner(config)
	if err == nil {
		t.Error("expected error from endpointTemplate of wrong type, got nil")
	}

	config["endpoint"] = []string{"biz", "boz", "baz"}
	_, err = NewInstanceRunner(config)
	if err == nil {
		t.Error("expected error from endpoint of wrong type, got nil")
	}

	expectedEndpointTemplate := "{{.Account}}"
	config["endpointTemplate"] = expectedEndpointTemplate
	_, err = NewInstanceRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set endpointTemplate, got %s", err)
	}

	expectedEndpoint := "some endpoint"
	config["endpoint"] = expectedEndpoint
	_, err = NewInstanceRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set endpoint, got %s", err)
	}

	expectedToken := "my-awesome-token"
	config["token"] = expectedToken

	out, err := NewInstanceRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set token, got %s", err)
	}

	if d := reflect.TypeOf(out).String(); d != "*jobs.InstanceRunner" {
		t.Errorf("expected type to be '*jobs.InstanceRunner', got '%s'", d)
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

func TestInstanceRunnerRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, "unexpected method "+r.Method)
			return
		}

		tok := r.Header.Get("Auth-Token")
		if tok != "my-awesome-token" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "bad token")
			return
		}

		if r.URL.String() == "/" {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, "OK")
		} else if r.URL.String() == "/myaccount/i-123456" {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, "KO")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "derp")
		}
	}))
	defer ts.Close()

	instanceRunner, err := NewInstanceRunner(map[string]interface{}{
		"token":    "my-awesome-token",
		"endpoint": ts.URL,
	})
	if err != nil {
		t.Errorf("expected nil error from set token, got %s", err)
	}

	if _, err = instanceRunner.Run(context.TODO(), "", "foo"); err == nil {
		t.Error("expected 'account is required' error for empty account, got nil")
	} else if err.Error() != "account is required" {
		t.Error("expected 'account is required' error for empty account, got nil")
	}

	if _, err = instanceRunner.Run(context.TODO(), "myaccount", map[string]string{}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected ErrMissingDetails with message 'missing instance_id', got %+v", e)
			}

			if e.Message != "missing instance_id" {
				t.Errorf("expected ErrMissingDetails with message 'missing instance_id', got %+v", e)
			}
		} else {
			t.Error("expected error for missing instance_id to be RunnerError")
		}
	} else {
		t.Error("expected error for missing instance_id, got nil")
	}

	if _, err = instanceRunner.Run(context.TODO(), "myaccount", map[string][]string{"instance_id": []string{"foo", "bar"}}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected ErrMissingDetails with message 'wrong type', got %+v", e)
			}

			if e.Message != "wrong type" {
				t.Errorf("expected ErrMissingDetails with message 'wrong type', got %+v", e)
			}
		} else {
			t.Error("expected error for missing instance_id to be RunnerError")
		}
	} else {
		t.Error("expected error for wrong instance_id type, got nil")
	}

	if _, err = instanceRunner.Run(context.TODO(), "myaccount", map[string]string{"instance_id": "i-123456"}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected ErrMissingDetails with message 'missing instance_action', got %+v", e)
			}

			if e.Message != "missing instance_action" {
				t.Errorf("expected ErrMissingDetails with message 'missing instance_action', got %+v", e)
			}
		} else {
			t.Error("expected error for missing instance_action to be RunnerError")
		}
	} else {
		t.Error("expected error for missing instance_action, got nil")
	}

	if _, err = instanceRunner.Run(context.TODO(), "myaccount", map[string]interface{}{
		"instance_id":     "i-123456",
		"instance_action": []string{"reboot", "stop", "start"},
	}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected ErrMissingDetails with message 'wrong type', got %+v", e)
			}

			if e.Message != "wrong type" {
				t.Errorf("expected ErrMissingDetails with message 'wrong type', got %+v", e)
			}
		} else {
			t.Error("expected error for wrong instance_action type to be RunnerError")
		}
	} else {
		t.Error("expected error for wrong instance_action type, got nil")
	}

	out, err := instanceRunner.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "i-123456",
		"instance_action": "stop",
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if out != "OK" {
		t.Errorf("expected OK, got %s", out)
	}

	_, err = instanceRunner.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "i-123456",
		"instance_action": "delete",
	})
	if err == nil {
		t.Error("expected error for bad action, got nil")
	}

	instanceRunnerTmpl, err := NewInstanceRunner(map[string]interface{}{
		"token":            "my-awesome-token",
		"endpointTemplate": fmt.Sprintf("%s/{{.Account}}/{{.InstanceID}}", ts.URL),
	})

	out, err = instanceRunnerTmpl.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "i-123456",
		"instance_action": "stop",
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if out != "KO" {
		t.Errorf("expected KO, got %s", out)
	}

	instanceRunnerBadURL, err := NewInstanceRunner(map[string]interface{}{
		"token":            "my-awesome-token",
		"endpointTemplate": fmt.Sprintf("%s/some-bad-url", ts.URL),
	})

	out, err = instanceRunnerBadURL.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "i-123456",
		"instance_action": "stop",
	})
	if err == nil {
		t.Error("expected error for bad URI, got nil")
	}
}
