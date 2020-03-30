package api

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/YaleSpinup/minion/cloudwatchlogs"
	"github.com/YaleSpinup/minion/common"
	"github.com/YaleSpinup/minion/jobs"
)

type mockCWLclient struct {
	t   *testing.T
	err error
}

type logGroup struct {
	name      string
	retention int64
	streams   map[string][]*cloudwatchlogs.Event
	tags      map[string]*string
}

var logGroupsMux sync.Mutex
var logGroups map[string]*logGroup

func (m *mockCWLclient) LogEvent(ctx context.Context, group, stream string, events []*cloudwatchlogs.Event) error {
	if m.err != nil {
		return m.err
	}

	for _, e := range events {
		m.t.Logf("logging event to %s/%s: %d %s", group, stream, e.Timestamp, e.Message)
	}

	// logging only unlocks mux when its done writing
	defer func() {
		m.t.Logf("unlocking log groups in LogEvent")
		logGroupsMux.Unlock()
	}()

	lg, ok := logGroups[group]
	if !ok {
		return errors.New("log group not found " + group)
	}

	logStream, ok := lg.streams[stream]
	if !ok {
		return errors.New("stream '" + stream + "' not found")
	}

	// append events to logs stream
	logStream = append(logStream, events...)
	lg.streams[stream] = logStream

	return nil
}

func (m *mockCWLclient) CreateLogGroup(ctx context.Context, group string, tags map[string]*string) error {
	if m.err != nil {
		return m.err
	}

	m.t.Logf("creating log group %s with tags %+v", group, tags)

	m.t.Log("locking log groups in CreateLogGroup")
	logGroupsMux.Lock()
	defer func() {
		m.t.Logf("unlocking log groups in CreateLogGroup")
		logGroupsMux.Unlock()
	}()

	if _, ok := logGroups[group]; ok {
		return errors.New("exists")
	}

	// create group
	logGroups[group] = &logGroup{
		name:    group,
		tags:    tags,
		streams: make(map[string][]*cloudwatchlogs.Event),
	}

	return nil
}

func (m *mockCWLclient) UpdateRetention(ctx context.Context, group string, retention int64) error {
	if m.err != nil {
		return m.err
	}

	m.t.Logf("updating log group %s retention to %d days", group, retention)

	m.t.Log("locking log groups in UpdateRetention")
	logGroupsMux.Lock()
	defer func() {
		m.t.Logf("unlocking log groups in UpdateRetention")
		logGroupsMux.Unlock()
	}()

	lg, ok := logGroups[group]
	if !ok {
		return errors.New("group not found " + group)
	}

	// set retention
	lg.retention = retention
	return nil
}

func (m *mockCWLclient) CreateLogStream(ctx context.Context, group, stream string) error {
	if m.err != nil {
		return m.err
	}

	m.t.Logf("creating log stream %s/%s", group, stream)

	m.t.Log("locking log groups in CreateLogStream")
	logGroupsMux.Lock()
	defer func() {
		m.t.Logf("unlocking log groups in CreateLogStream")
		logGroupsMux.Unlock()
	}()

	lg, ok := logGroups[group]
	if !ok {
		return errors.New("group not found: " + group)
	}

	if _, ok := lg.streams[stream]; ok {
		return errors.New("stream already exists in group")
	}

	// create stream
	lg.streams[stream] = []*cloudwatchlogs.Event{}
	return nil
}

func newMockLogger(prefix string, timeout time.Duration, cwl *mockCWLclient) *logger {
	return &logger{
		client:  cwl,
		prefix:  prefix,
		timeout: timeout,
	}
}

func TestNewLogger(t *testing.T) {
	input := common.LogProvider{
		Region: "sanitarium",
		Akid:   "welcome-home",
		Secret: "masterofpuppets1986",
	}

	l := newLogger("foo", input)
	if is := reflect.TypeOf(l).String(); is != "*api.logger" {
		t.Errorf("expected newLogger to return '*api.logger', got %s", is)
	}

	if l.prefix != "foo" {
		t.Errorf("expected prefix to be 'foo' got %s", l.prefix)
	}

	if l.timeout != 5*time.Minute {
		t.Errorf("expected timeout to be 5 minutes, got %s", l.timeout.String())
	}
}

func TestCreateLog(t *testing.T) {
	logGroups = make(map[string]*logGroup)
	l := newMockLogger("test", 5*time.Second, &mockCWLclient{t: t})

	tags := []*jobs.Tag{
		&jobs.Tag{Key: "soClose", Value: "noMatterHowFar"},
		&jobs.Tag{Key: "couldntBe", Value: "muchMoreFromTheHeart"},
		&jobs.Tag{Key: "forever", Value: "trustingWhoWeAre"},
		&jobs.Tag{Key: "andNothing", Value: "elseMatters"},
	}

	expectedTags := make(map[string]*string)
	for _, tag := range tags {
		expectedTags[tag.Key] = &tag.Value
	}

	expected := &logGroup{
		name:      "test-group",
		retention: int64(90),
		streams: map[string][]*cloudwatchlogs.Event{
			"test-stream": []*cloudwatchlogs.Event{},
		},
		tags: expectedTags,
	}

	if err := l.createLog(context.TODO(), "group", "test-stream", tags); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if lg, ok := logGroups["test-group"]; !ok {
		t.Error("expected log group 'test-group' to exist")
	} else {
		if !reflect.DeepEqual(lg, expected) {
			t.Errorf("expected %+v, got %+v", expected, lg)
		}
	}

	l = newMockLogger("test", 5*time.Second, &mockCWLclient{t: t, err: errors.New("boom!")})
	if err := l.createLog(context.TODO(), "nonexistent-group", "test-stream", tags); err == nil {
		t.Error("expected error for missing log-group, got nil")
	}
}

func TestLog(t *testing.T) {
	logGroups = make(map[string]*logGroup)
	testLogGroup := logGroup{
		name:      "test-group",
		retention: int64(365),
		streams: map[string][]*cloudwatchlogs.Event{
			"test-stream": []*cloudwatchlogs.Event{},
		},
	}
	logGroups = map[string]*logGroup{
		testLogGroup.name: &testLogGroup,
	}

	testMessages := []string{
		"some random message",
		"some random message",
		"some random message",
		"some random message",
		"some random message",
	}

	logGroupsMux.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	messageStream := newMockLogger("test", 5*time.Second, &mockCWLclient{t: t}).log(ctx, "test-group", "test-stream")
	for _, m := range testMessages {
		messageStream <- m
	}
	cancel()

	// attempt to lock so we know the messages were written to the map
	logGroupsMux.Lock()
	for _, lg := range logGroups {
		t.Logf("log-group: %+v", lg)
	}

	s, ok := testLogGroup.streams["test-stream"]
	if !ok {
		t.Errorf("expected log stream 'test-stream' to exist")
	}

	resultMessages := []string{}
	for _, m := range s {
		resultMessages = append(resultMessages, m.Message)
	}

	if !reflect.DeepEqual(testMessages, resultMessages) {
		t.Errorf("expected: %+v, got %+v", testMessages, resultMessages)
	}
	logGroupsMux.Unlock()
}
