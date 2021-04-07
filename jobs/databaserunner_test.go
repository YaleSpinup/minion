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

func TestNewDatabaseRunner(t *testing.T) {
	_, err := NewDatabaseRunner(map[string]interface{}{})
	if err == nil {
		t.Error("expected error from empty configuration, got nil")
	}

	config := map[string]interface{}{
		"foo": "bar",
	}
	_, err = NewDatabaseRunner(config)
	if err == nil {
		t.Error("expected error from missing endpoint and endpointTemplate, got nil")
	}

	config["endpointTemplate"] = []string{"biz", "boz", "baz"}
	_, err = NewDatabaseRunner(config)
	if err == nil {
		t.Error("expected error from endpointTemplate of wrong type, got nil")
	}

	config["endpoint"] = []string{"biz", "boz", "baz"}
	_, err = NewDatabaseRunner(config)
	if err == nil {
		t.Error("expected error from endpoint of wrong type, got nil")
	}

	expectedEndpointTemplate := "{{.Account}}"
	config["endpointTemplate"] = expectedEndpointTemplate
	_, err = NewDatabaseRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set endpointTemplate, got %s", err)
	}

	expectedEndpoint := "some endpoint"
	config["endpoint"] = expectedEndpoint
	_, err = NewDatabaseRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set endpoint, got %s", err)
	}

	expectedToken := "my-awesome-token"
	config["token"] = expectedToken

	out, err := NewDatabaseRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set token, got %s", err)
	}

	if d := reflect.TypeOf(out).String(); d != "*jobs.DatabaseRunner" {
		t.Errorf("expected type to be '*jobs.DatabaseRunner', got '%s'", d)
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

func TestDatabaseRunnerRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, "unexpected method "+r.Method)
			return
		}

		tok := r.Header.Get("X-Auth-Token")
		if tok != "my-awesome-token" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "bad token")
			return
		}

		if r.URL.String() == "/" {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, "OK")
		} else if r.URL.String() == "/myaccount/db-123456/power" {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, "KO")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "derp")
		}
	}))
	defer ts.Close()

	databaseRunner, err := NewDatabaseRunner(map[string]interface{}{
		"token":    "my-awesome-token",
		"endpoint": ts.URL,
	})
	if err != nil {
		t.Errorf("expected nil error from set token, got %s", err)
	}

	if _, err = databaseRunner.Run(context.TODO(), "", "foo"); err == nil {
		t.Error("expected 'account is required' error for empty account, got nil")
	} else if err.Error() != "account is required" {
		t.Error("expected 'account is required' error for empty account, got nil")
	}

	if _, err = databaseRunner.Run(context.TODO(), "myaccount", map[string]string{}); err != nil {
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

	if _, err = databaseRunner.Run(context.TODO(), "myaccount", map[string][]string{"instance_id": {"foo", "bar"}}); err != nil {
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

	if _, err = databaseRunner.Run(context.TODO(), "myaccount", map[string]string{"instance_id": "db-123456"}); err != nil {
		e := RunnerError{}
		if errors.As(err, &e) {
			if e.Code != ErrMissingDetails {
				t.Errorf("expected ErrMissingDetails with message 'missing database_action', got %+v", e)
			}

			if e.Message != "missing database_action" {
				t.Errorf("expected ErrMissingDetails with message 'missing database_action', got %+v", e)
			}
		} else {
			t.Error("expected error for missing database_action to be RunnerError")
		}
	} else {
		t.Error("expected error for missing database_action, got nil")
	}

	if _, err = databaseRunner.Run(context.TODO(), "myaccount", map[string]interface{}{
		"instance_id":     "db-123456",
		"database_action": []string{"stop", "start"},
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
			t.Error("expected error for wrong database_action type to be RunnerError")
		}
	} else {
		t.Error("expected error for wrong database_action type, got nil")
	}

	out, err := databaseRunner.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "db-123456",
		"database_action": "stop",
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if out != "OK" {
		t.Errorf("expected OK, got %s", out)
	}

	_, err = databaseRunner.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "db-123456",
		"database_action": "delete",
	})
	if err == nil {
		t.Error("expected error for bad action, got nil")
	}

	databaseRunnerTmpl, err := NewDatabaseRunner(map[string]interface{}{
		"token":            "my-awesome-token",
		"endpointTemplate": fmt.Sprintf("%s/{{.Account}}/{{.InstanceID}}/power", ts.URL),
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	out, err = databaseRunnerTmpl.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "db-123456",
		"database_action": "stop",
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if out != "KO" {
		t.Errorf("expected KO, got %s", out)
	}

	databaseRunnerBadURL, err := NewDatabaseRunner(map[string]interface{}{
		"token":            "my-awesome-token",
		"endpointTemplate": fmt.Sprintf("%s/some-bad-url", ts.URL),
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	_, err = databaseRunnerBadURL.Run(context.TODO(), "myaccount", map[string]string{
		"instance_id":     "db-123456",
		"database_action": "stop",
	})
	if err == nil {
		t.Error("expected error for bad URI, got nil")
	}
}
