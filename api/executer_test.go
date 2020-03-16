package api

import (
	"context"
	"errors"
	"testing"

	"github.com/YaleSpinup/minion/jobs"
	"github.com/google/uuid"
)

type mockExecQueuer struct {
	t         *testing.T
	finalize  bool
	finalized bool
}

func newMockExecQueuer(t *testing.T, finalize bool) *mockExecQueuer {
	return &mockExecQueuer{
		t:         t,
		finalize:  finalize,
		finalized: false,
	}
}

func (m *mockExecQueuer) Close() error {
	m.t.Log("closing executer queuer")
	return nil
}
func (m *mockExecQueuer) Enqueue(queued *jobs.QueuedJob) error {
	m.t.Logf("executer enqueing job %+v", queued)
	return nil
}
func (m *mockExecQueuer) Fetch(queued *jobs.QueuedJob) error {
	m.t.Log("executer fetching jobs")
	return nil
}
func (m *mockExecQueuer) Finalize(id string) error {
	m.t.Logf("executer finalizing job %s", id)

	if !m.finalize {
		m.t.Logf("'finalize' set to false, not finalizing")
		return errors.New("boom!")
	}

	m.finalized = true
	return nil
}

type mockRunner struct {
	t            *testing.T
	succeedafter int
	count        int
	ran          bool
}

func newMockRunner(t *testing.T, succeedAfter int) *mockRunner {
	return &mockRunner{
		t:            t,
		succeedafter: succeedAfter,
		count:        0,
		ran:          false,
	}
}

func (m *mockRunner) Run(ctx context.Context, account string, parameters interface{}) (string, error) {
	m.count += 1

	if m.succeedafter < m.count {
		m.t.Logf("run count number %d", m.count)
		m.ran = true
		return "success", nil
	}

	m.t.Logf("failing on count %d", m.count)
	return "", errors.New("boom")
}

func newMockExecuter(t *testing.T, q *mockExecQueuer) *executer {
	q.t = t
	q.finalized = false
	return &executer{
		id:       uuid.New().String(),
		jobQueue: q,
	}
}

func TestExecuterRun(t *testing.T) {
	// test early success.  always finalize, test runner ran
	q := newMockExecQueuer(t, true)
	r := newMockRunner(t, 0)
	newMockExecuter(t, q).run(context.TODO(), r, &jobs.Job{ID: "job1"})
	if !q.finalized {
		t.Error("queue was not finalized")
	}

	if !r.ran {
		t.Error("runner didn't run, expected runner to run")
	}

	// test no success (retry is 3).  always finalize, test runner didn't run
	q = newMockExecQueuer(t, true)
	r = newMockRunner(t, 5)
	newMockExecuter(t, q).run(context.TODO(), r, &jobs.Job{ID: "job2"})
	if !q.finalized {
		t.Error("queue was not finalized")
	}

	if r.ran {
		t.Error("runner ran, expected no run for failures > 3")
	}

	// test early success.  failed finalize
	q = newMockExecQueuer(t, false)
	r = newMockRunner(t, 0)
	q.finalize = false
	newMockExecuter(t, q).run(context.TODO(), r, &jobs.Job{ID: "job1"})
	if q.finalized {
		t.Error("queue was not finalized")
	}

	if !r.ran {
		t.Error("runner didn't run, expected runner to run")
	}
}
