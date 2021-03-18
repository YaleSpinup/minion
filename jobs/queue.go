package jobs

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Queuer interface {
	Close() error
	Enqueue(queued *QueuedJob) error
	Fetch(queued *QueuedJob) error
	Finalize(id string) error
}

type QueuedJob struct {
	ID    string
	Score float64
}

type RedisQueuer struct {
	BackupName string
	client     *redis.Client
	Name       string
	Window     int64
}

func NewRedisQueuer(name, address, password string, db int, window int64) (*RedisQueuer, error) {
	return &RedisQueuer{
		Name:       name,
		BackupName: name + "-backup",
		Window:     window,
		client: redis.NewClient(&redis.Options{
			Addr:       address,
			PoolSize:   5,
			MaxRetries: 2,
			Password:   password,
			DB:         db,
		}),
	}, nil

}

func (q *RedisQueuer) Fetch(queued *QueuedJob) error {
	val, err := q.client.BZPopMin(2*time.Second, q.Name).Result()
	if err != nil {
		if err == redis.Nil {
			return NewQueueError(ErrQueueIsEmpty, "redis queue is empty", err)
		}

		log.Warnf("Error in getting job: %s", err)
		return err
	}

	log.Debugf("got value from bzpop: %+v", val)

	id, ok := val.Member.(string)
	if !ok {
		return fmt.Errorf("unexpected member value for queued job, not a string: %+v", val.Member)
	}

	log.Debugf("current time: %d, time of job: %f, job id: %s", currentTime(), val.Score, id)

	queued.ID = id
	queued.Score = val.Score

	// if the queued score (requested execution) minus the current time is greater than the allowed window,
	// the job is supposed to execute too far in the future, so reschedule and return an error.
	if int64(val.Score)-currentTime() > q.Window {
		log.Debugf("job '%s' is not within the window, rescheduling", id)
		if err := q.enqueue(q.Name, queued.Score, queued.ID); err != nil {
			log.Errorf("failed to re-enqueue job: %s", err)
		}
		return fmt.Errorf("rescheduled job, not within window")
	}

	return nil
}

// Add jobs to both sets
func (q *RedisQueuer) Enqueue(queued *QueuedJob) error {
	log.Debugf("enqueuing job %s", queued.ID)

	if err := q.enqueue(q.Name, queued.Score, queued.ID); err != nil {
		return err
	}

	if err := q.enqueue(q.BackupName, queued.Score, queued.ID); err != nil {
		return err
	}

	return nil
}

// Finalize does the final steps once a job is completed successfully, currently this
// is just dequeuing the backup job created when the job was queued.
func (q *RedisQueuer) Finalize(id string) error {
	log.Debugf("finalizing job %s", id)

	if err := q.dequeue(q.BackupName, id); err != nil {
		return err
	}
	return nil
}

// Close the redis client connection
func (q *RedisQueuer) Close() error {
	return q.client.Close()
}

func (q *RedisQueuer) dequeue(setName string, id string) error {
	if err := q.client.ZRem(setName, id).Err(); err != nil {
		return errors.Wrap(err, "failed removing job "+id)
	}

	return nil
}

func (q *RedisQueuer) enqueue(setName string, timestamp float64, id string) error {
	if err := q.client.ZAdd(setName, redis.Z{
		Score:  float64(timestamp),
		Member: id,
	}).Err(); err != nil {
		return errors.Wrap(err, "failed adding job "+id)
	}
	return nil
}

func currentTime() int64 {
	return (time.Now().UnixNano() / (int64(time.Second) / int64(time.Nanosecond)))
}
