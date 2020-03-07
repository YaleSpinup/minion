package jobs

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewQueueError(t *testing.T) {
	out := NewQueueError(ErrQueueIsEmpty, "queue is empty", errors.New("fail"))
	if e := reflect.TypeOf(out).String(); e != "jobs.QueueError" {
		t.Errorf("expect type to be apierror.Error, got %s", e)
	}
}

func TestQueueErrorError(t *testing.T) {
	out := NewQueueError(ErrQueueIsEmpty, "queue is empty", errors.New("fail"))
	if out.Error() != out.String() {
		t.Errorf("expected '%s', got '%s'", out.String(), out)
	}

	out = NewQueueError(ErrQueueIsEmpty, "queue is empty", errors.New("fail"))
	if out.Error() != out.String() {
		t.Errorf("expected '%s', got '%s'", out.String(), out)
	}
}

func TestQueueErrorString(t *testing.T) {
	out := NewQueueError(ErrQueueIsEmpty, "queue is empty", errors.New("fail"))
	expect := "QueueIsEmpty: queue is empty (fail)"
	if out.String() != expect {
		t.Errorf("expected '%s', got '%s'", expect, out)
	}

	out = NewQueueError(ErrQueueIsEmpty, "queue is empty", nil)
	expect = "QueueIsEmpty: queue is empty"
	if out.String() != expect {
		t.Errorf("expected '%s', got '%s'", expect, out)
	}
}

func TestQueueErrorUnwrap(t *testing.T) {
	err := errors.New("Fail")

	out := NewQueueError(ErrQueueIsEmpty, "queue is empty", err)
	orig := out.Unwrap()

	if orig != err {
		t.Errorf("expect original error tp be %s, got %s", err, orig)
	}
}
