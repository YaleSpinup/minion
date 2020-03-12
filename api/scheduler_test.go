package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YaleSpinup/minion/jobs"
	"github.com/google/uuid"
)

var schedTestJobCache = jobsCache{
	Cache: map[string]*jobs.Job{
		"good-schedule-expression-1": &jobs.Job{
			Description:        "enqueue me every minute",
			ScheduleExpression: "* * * * *",
		},
		"good-schedule-expression-2": &jobs.Job{
			Description:        "enqueue me hourly",
			ScheduleExpression: "@hourly",
		},
		"good-schedule-expression-3": &jobs.Job{
			Description:        "enqueue me every five",
			ScheduleExpression: "*/5 * * * *",
		},
		"bad-schedule-expression": &jobs.Job{
			Description:        "im broke, dont queue me",
			ScheduleExpression: "broke",
		},
	},
}

type mockSchedLocker struct {
	t    *testing.T
	lock bool
}

func (m *mockSchedLocker) Lock(key, id string) error {
	m.t.Logf("lock got key %s, id %s", key, id)
	if m.lock {
		return nil
	}
	return errors.New("nope")
}

type mockSchedQueuer struct {
	t     *testing.T
	queue bool
}

func (m *mockSchedQueuer) Close() error {
	m.t.Log("closing schedule queuer")
	return nil
}
func (m *mockSchedQueuer) Enqueue(queued *jobs.QueuedJob) error {
	m.t.Logf("enqueing job %+v", queued)

	if queued.ID == "bad-schedule-expression" {
		m.t.Errorf("bad-schedule-expression shouldn't get queued")
	}

	if !m.queue {
		m.t.Errorf("job shouldn't be queued %+v", queued)
	}

	return nil
}
func (m *mockSchedQueuer) Fetch(queued *jobs.QueuedJob) error {
	m.t.Log("fetching jobs")
	return nil
}
func (m *mockSchedQueuer) Finalize(id string) error {
	m.t.Logf("finalizing job %s", id)
	return nil
}

func TestSchedulerRun(t *testing.T) {
	id := uuid.New().String()
	sched := &scheduler{
		id:        id,
		jobsCache: &schedTestJobCache,
		jobQueue:  &mockSchedQueuer{t, true},
		locker:    &mockSchedLocker{t, true},
	}

	nowBasis := time.Now().UTC().Truncate(time.Minute)
	minuteBasis, _ := time.Parse(time.RFC3339, "2020-03-13T01:03:00.123Z")
	hourBasis, _ := time.Parse(time.RFC3339, "2020-03-13T01:00:00.123Z")
	fiveMinutesBasis, _ := time.Parse(time.RFC3339, "2020-03-13T01:05:00.123Z")

	// test happy path
	sched.run(context.TODO(), nowBasis)
	sched.run(context.TODO(), minuteBasis)
	sched.run(context.TODO(), hourBasis)
	sched.run(context.TODO(), fiveMinutesBasis)

	// test false locks
	sched.locker = &mockSchedLocker{t, false}
	sched.jobQueue = &mockSchedQueuer{t, false}
	sched.run(context.TODO(), nowBasis)
	sched.run(context.TODO(), minuteBasis)
	sched.run(context.TODO(), hourBasis)
	sched.run(context.TODO(), fiveMinutesBasis)
}
