package jobs

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
)

type Locker interface {
	Lock(key, id string) error
}

// RedisLocker is a redis lock/unlock provider.
type RedisLocker struct {
	client *redis.Client
	Expire time.Duration
	Prefix string
}

// NewRedisLocker returns a new redis lock provider
func NewRedisLocker(prefix, address, password string, db int, expiration string) (*RedisLocker, error) {
	exp, err := time.ParseDuration(expiration)
	if err != nil {
		return nil, err
	}

	return &RedisLocker{
		Prefix: prefix,
		client: redis.NewClient(&redis.Options{
			Addr:       address,
			PoolSize:   5,
			MaxRetries: 2,
			Password:   password,
			DB:         db,
		}),
		Expire: exp,
	}, nil
}

// Lock locks l.Prefix-key with the value id in a redis set. This uses SetNX.  If the result is an error
// or false, the key was not set and the lock was not aquired.
func (l *RedisLocker) Lock(key, id string) error {
	key = l.Prefix + "-" + key

	out, err := l.client.SetNX(key, id, l.Expire).Result()
	if err != nil {
		return err
	}

	if !out {
		return errors.New("didn't aquire lock")
	}

	return nil
}
