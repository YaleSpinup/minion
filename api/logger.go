package api

import (
	"context"
	"time"

	"github.com/YaleSpinup/minion/cloudwatchlogs"
	"github.com/YaleSpinup/minion/common"
	"github.com/YaleSpinup/minion/jobs"
	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
)

type cwlogsIface interface {
	LogEvent(ctx context.Context, group, stream string, events []*cloudwatchlogs.Event) error
	CreateLogGroup(ctx context.Context, group string, tags map[string]*string) error
	UpdateRetention(ctx context.Context, group string, retention int64) error
	CreateLogStream(ctx context.Context, group, stream string) error
}

type logger struct {
	client  cwlogsIface
	prefix  string
	timeout time.Duration
}

func newLogger(org string, config common.LogProvider) *logger {
	cwClient := cloudwatchlogs.NewSession(config.Region, config.Akid, config.Secret)
	return &logger{
		client:  &cwClient,
		prefix:  org,
		timeout: 5 * time.Minute,
	}
}

func (l *logger) log(ctx context.Context, group, stream string) chan string {
	messageStream := make(chan string)

	// TODO: this will fail if there are more than 10,000 entries batched.  Initially, I
	// handled this case, but I don't think we'll ever need it (and we can add the complexity
	// then if we do).  Removing the logic, makes this much simpler.
	go func() {
		log.Debugf("starting log batching go routine")

		// default to 10 minutes
		timeout := 10 * time.Minute
		if l.timeout != 0 {
			timeout = l.timeout
		}

		messages := []*cloudwatchlogs.Event{}

		defer func() {
			log.Debug("finalizing log batch")

			logctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if len(messages) > 0 {
				for _, m := range messages {
					log.Debugf("sending log event to %s/%s: %d %s", group, stream, m.Timestamp, m.Message)
				}

				if err := l.client.LogEvent(logctx, group, stream, messages); err != nil {
					log.Errorf("failed to log events: %s", err)
				}
			}
		}()

		for {
			log.Debug("starting log batch collection loop")
			select {
			case message := <-messageStream:
				timestamp := time.Now().UnixNano() / int64(time.Millisecond)
				log.Debugf("%d received message %s", timestamp, message)
				messages = append(messages, &cloudwatchlogs.Event{
					Message:   message,
					Timestamp: timestamp,
				})
			case <-time.After(timeout):
				log.Warnf("timed out waiting for more log messages to write to %s/%s", group, stream)
				return
			case <-ctx.Done():
				log.Debug("context closed")
				return
			}
		}
	}()

	return messageStream
}

func (l *logger) createLog(ctx context.Context, group, stream string, tags []*jobs.Tag) error {
	var tagsMap = make(map[string]*string)
	for _, tag := range tags {
		tagsMap[tag.Key] = aws.String(tag.Value)
	}

	logGroup := group
	if l.prefix != "" {
		logGroup = l.prefix + "-" + logGroup
	}

	if err := l.client.CreateLogGroup(ctx, logGroup, tagsMap); err != nil {
		return err
	}

	if err := l.client.UpdateRetention(ctx, logGroup, int64(90)); err != nil {
		return err
	}

	if err := l.client.CreateLogStream(ctx, logGroup, stream); err != nil {
		return err
	}

	return nil
}
