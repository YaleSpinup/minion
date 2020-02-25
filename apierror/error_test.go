package apierror

import (
	"errors"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	out := New(ErrBadRequest, "bad request", errors.New("fail"))
	if e := reflect.TypeOf(out).String(); e != "apierror.Error" {
		t.Errorf("expect type to be apierror.Error, got %s", e)
	}
}

func TestError(t *testing.T) {
	out := New(ErrBadRequest, "bad request", errors.New("fail"))
	if out.Error() != out.String() {
		t.Errorf("expected '%s', got '%s'", out.String(), out)
	}

	out = New(ErrBadRequest, "bad request", nil)
	if out.Error() != out.String() {
		t.Errorf("expected '%s', got '%s'", out.String(), out)
	}
}

func TestString(t *testing.T) {
	out := New(ErrBadRequest, "bad request", errors.New("fail"))
	expect := "BadRequest: bad request (fail)"
	if out.String() != expect {
		t.Errorf("expected '%s', got '%s'", expect, out)
	}

	out = New(ErrBadRequest, "bad request", nil)
	expect = "BadRequest: bad request"
	if out.String() != expect {
		t.Errorf("expected '%s', got '%s'", expect, out)
	}
}

func TestUnwrap(t *testing.T) {
	err := errors.New("Fail")

	out := New(ErrBadRequest, "bad request", err)
	orig := out.Unwrap()

	if orig != err {
		t.Errorf("expect original error tp be %s, got %s", err, orig)
	}
}
