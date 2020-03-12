package jobs

import(
	"testing"
	"reflect"
)

func TestNewRedisQueuer(t *testing.T) {
	r, err := NewRedisQueuer("foo", "127.0.0.1:6379", "", 0, 10)
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if to := reflect.TypeOf(r).String(); to != "*jobs.RedisQueuer" {
		t.Errorf("expected type to be '*jobs.RedisQueuer, got %s", to)
	}
}