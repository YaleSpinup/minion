package jobs

import (
	"context"
	"reflect"
	"testing"
)

func TestNewDummyRunner(t *testing.T) {
	_, err := NewDummyRunner(map[string]interface{}{})
	if err == nil {
		t.Error("expected error from empty configuration, got nil")
	}

	config := map[string]interface{}{
		"foo": "bar",
	}
	_, err = NewDummyRunner(config)
	if err == nil {
		t.Error("expected error from missing template, got nil")
	}

	config["template"] = []string{"biz", "boz", "baz"}
	_, err = NewDummyRunner(config)
	if err == nil {
		t.Error("expected error from template of wrong type, got nil")
	}

	config["template"] = "my awesome template"
	out, err := NewDummyRunner(config)
	if err != nil {
		t.Errorf("expected nil error from set template, got %s", err)
	}

	if d := reflect.TypeOf(out).String(); d != "*jobs.DummyRunner" {
		t.Errorf("expected type to be '*jobs.DummyRunner', got '%s'", d)
	}
}

func TestDummyRunnerRun(t *testing.T) {
	dummyRunner, err := NewDummyRunner(map[string]interface{}{
		"template": "Hi, {{.Account}}!",
	})
	if err != nil {
		t.Errorf("expected nil error from new dummyrunner, got %s", err)
	}

	_, err = dummyRunner.Run(context.TODO(), "", "foo")
	if err == nil {
		t.Error("expected error for empty account, got nil")
	}

	out, err := dummyRunner.Run(context.TODO(), "myaccount", "foo")
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if out != "Hi, myaccount!" {
		t.Errorf("expected 'Hi, myaccount', got '%s'", out)
	}

	badDummyRunner, err := NewDummyRunner(map[string]interface{}{
		"template": "Hi, {{.Blarg}}!",
	})
	if err != nil {
		t.Errorf("expected nil error from new dummyrunner, got %s", err)
	}

	_, err = badDummyRunner.Run(context.TODO(), "myaccount", "foo")
	if err == nil {
		t.Error("expected error for bad template, got nil")
	}
}
