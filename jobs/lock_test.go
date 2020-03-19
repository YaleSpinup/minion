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

	_, err = NewRedisLocker("foo", "127.0.0.1:6379", "", 0, "somebadduration")
	if err == nil || err.Error() != "time: invalid duration somebadduration" {
		t.Error("expected error 'time: invalid duration somebadduration', got nil")
	}
}
