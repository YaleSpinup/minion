package jobs

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Queue struct {
	Name       string
	BackupName string
	Window     int64
	client     *redis.Client
}

type QueuedJob struct {
	ID    string
	Score float64
}

func NewQueue(name, address, password string, db int, window int64) (*Queue, error) {
	return &Queue{
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

func (q *Queue) Fetch(queued *QueuedJob) error {
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
		q.enqueue(q.Name, queued.Score, queued.ID)
		return fmt.Errorf("rescheduled job, not within window")
	}

	return nil
}

// Add jobs to both sets
func (q *Queue) Enqueue(queued *QueuedJob) error {
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
func (q *Queue) Finalize(id string) error {
	log.Debugf("finalizing job %s", id)

	if err := q.dequeue(q.BackupName, id); err != nil {
		return err
	}
	return nil
}

// Close the redis client connection
func (q *Queue) Close() error {
	return q.client.Close()
}

func (q *Queue) dequeue(setName string, id string) error {
	if err := q.client.ZRem(setName, id).Err(); err != nil {
		return errors.Wrap(err, "failed removing job "+id)
	}

	return nil
}

func (q *Queue) enqueue(setName string, timestamp float64, id string) error {
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
