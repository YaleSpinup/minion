package jobs

import (
	"errors"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	out := NewRunnerError(ErrExecFailure, "boom", errors.New("fail"))
	if e := reflect.TypeOf(out).String(); e != "jobs.RunnerError" {
		t.Errorf("expect type to be jobs.RunnerError, got %s", e)
	}
}

func TestError(t *testing.T) {
	out := NewRunnerError(ErrExecFailure, "boom", errors.New("fail"))
	if out.Error() != out.String() {
		t.Errorf("expected '%s', got '%s'", out.String(), out)
	}

	out = NewRunnerError(ErrExecFailure, "boom", nil)
	if out.Error() != out.String() {
		t.Errorf("expected '%s', got '%s'", out.String(), out)
	}
}

func TestString(t *testing.T) {
	out := NewRunnerError(ErrExecFailure, "boom", errors.New("fail"))
	expect := "ExecutionFailure: boom (fail)"
	if out.String() != expect {
		t.Errorf("expected '%s', got '%s'", expect, out)
	}

	out = NewRunnerError(ErrExecFailure, "boom", nil)
	expect = "ExecutionFailure: boom"
	if out.String() != expect {
		t.Errorf("expected '%s', got '%s'", expect, out)
	}
}

func TestUnwrap(t *testing.T) {
	err := errors.New("Fail")

	out := NewRunnerError(ErrExecFailure, "boom", err)
	orig := out.Unwrap()

	if orig != err {
		t.Errorf("expect original error tp be %s, got %s", err, orig)
	}
}
