package jobs

import (
	"reflect"
	"testing"
)

func TestNewRedisLocker(t *testing.T) {
	r, err := NewRedisLocker("foo", "127.0.0.1:6379", "", 0, "10s")
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if to := reflect.TypeOf(r).String(); to != "*jobs.RedisLocker" {
		t.Errorf("expected type to be '*jobs.RedisLocker, got %s", to)
	}

	if _, err = NewRedisLocker("foo", "127.0.0.1:6379", "", 0, "somebadduration"); err == nil {
		t.Error("expected error got nil")
	} else if err.Error() != "time: invalid duration \"somebadduration\"" {
		t.Errorf("expected error 'time: invalid duration \"somebadduration\"', got %s", err)
	}
}
