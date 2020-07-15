package cloudwatchlogs

import (
	"context"
	"fmt"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	log "github.com/sirupsen/logrus"
)

// CloudWatchLogs is the internal cloudwatch logsobject which holds session
// and configuration information
type CloudWatchLogs struct {
	Service cloudwatchlogsiface.CloudWatchLogsAPI
}

// Event is a cloudwatchlogs Event
type Event struct {
	Message   string
	Timestamp int64
}

// LogGroup is a cloudwatchlogs log group
type LogGroup struct {
	Name      *string   `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Retention *int64    `json:"retention"`
	Bytes     *int64    `json:"bytes"`
}

// NewSession builds a new aws cloudwatchlogs session
func NewSession(region, akid, secret string) CloudWatchLogs {
	c := CloudWatchLogs{}
	log.Infof("Creating new session with key id %s in region %s", akid, region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(akid, secret, ""),
		Region:      aws.String(region),
	}))
	c.Service = cloudwatchlogs.New(sess)
	return c
}

func (c *CloudWatchLogs) GetLogEvents(ctx context.Context, input *cloudwatchlogs.GetLogEventsInput) (*cloudwatchlogs.GetLogEventsOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	output, err := c.Service.GetLogEventsWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to get log events", err)
	}

	return output, nil
}

// GetLogGroupTags returns the list of tags on a log group
func (c *CloudWatchLogs) GetLogGroupTags(ctx context.Context, group string) (map[string]*string, error) {
	if group == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	out, err := c.Service.ListTagsLogGroupWithContext(ctx, &cloudwatchlogs.ListTagsLogGroupInput{LogGroupName: aws.String(group)})
	if err != nil {
		msg := fmt.Sprintf("failed to list tags for log group (%s)", group)
		return nil, ErrCode(msg, err)
	}

	return out.Tags, nil
}

// TagLogGroup sets tags for a log group
func (c *CloudWatchLogs) TagLogGroup(ctx context.Context, group string, tags map[string]*string) error {
	if group == "" || tags == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	if _, err := c.Service.TagLogGroupWithContext(ctx, &cloudwatchlogs.TagLogGroupInput{
		LogGroupName: aws.String(group),
		Tags:         tags,
	}); err != nil {
		msg := fmt.Sprintf("failed to set tags for log group (%s)", group)
		return ErrCode(msg, err)
	}

	return nil
}

// DescribeLogGroup describes a cloudwatchlogs log group
func (c *CloudWatchLogs) DescribeLogGroup(ctx context.Context, group string) (*LogGroup, error) {
	if group == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	out, err := c.Service.DescribeLogGroupsWithContext(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(group),
	})
	if err != nil {
		msg := fmt.Sprintf("failed to describe log group (%s)", group)
		return nil, ErrCode(msg, err)
	}

	var logGroup *LogGroup
	for _, lg := range out.LogGroups {
		if aws.StringValue(lg.LogGroupName) == group {
			logGroup = &LogGroup{
				Name:      lg.LogGroupName,
				CreatedAt: aws.MillisecondsTimeValue(lg.CreationTime),
				Retention: lg.RetentionInDays,
				Bytes:     lg.StoredBytes,
			}
		}
	}

	if logGroup == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "log group doesn't exist", nil)
	}

	return logGroup, nil
}

// CreateLogGroup creates a cloudwatchlogs log group
func (c *CloudWatchLogs) CreateLogGroup(ctx context.Context, group string, tags map[string]*string) error {
	if group == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	if _, err := c.Service.CreateLogGroupWithContext(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(group),
		Tags:         tags,
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case cloudwatchlogs.ErrCodeResourceAlreadyExistsException:
				log.Warnf("cloudwatch log group (%s) already exists, continuing: (%s)", group, err)
			default:
				msg := fmt.Sprintf("failed to create log group (%s)", group)
				return ErrCode(msg, err)
			}
		}
	}

	return nil
}

// DeleteLogGroup deletes a cloudwatchlogs log group
func (c *CloudWatchLogs) DeleteLogGroup(ctx context.Context, group string) error {
	if group == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	if _, err := c.Service.DeleteLogGroupWithContext(ctx, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: aws.String(group),
	}); err != nil {
		msg := fmt.Sprintf("failed to delete log group (%s)", group)
		return ErrCode(msg, err)
	}

	return nil
}

// UpdateRetention changes the retention (in days) for logs in a log group
func (c *CloudWatchLogs) UpdateRetention(ctx context.Context, group string, retention int64) error {
	if group == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	_, err := c.Service.PutRetentionPolicyWithContext(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(group),
		RetentionInDays: aws.Int64(retention),
	})
	if err != nil {
		msg := fmt.Sprintf("failed to update retention policy for log group (%s)", group)
		return ErrCode(msg, err)
	}

	return nil
}

// CreateLogStream creates a cloudwatchlogs log stream
func (c *CloudWatchLogs) CreateLogStream(ctx context.Context, group, stream string) error {
	if group == "" || stream == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	if _, err := c.Service.CreateLogStreamWithContext(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case cloudwatchlogs.ErrCodeResourceAlreadyExistsException:
				log.Warnf("cloudwatch log stream (%s/%s) already exists, continuing: (%s)", group, stream, err)
			default:
				msg := fmt.Sprintf("failed to create log stream (%s/%s)", group, stream)
				return ErrCode(msg, err)
			}
		}
	}

	return nil
}

// LogEvent logs events to a log stream in a log group
func (c *CloudWatchLogs) LogEvent(ctx context.Context, group, stream string, events []*Event) error {
	if group == "" || stream == "" || len(events) == 0 {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	lsOut, err := c.Service.DescribeLogStreamsWithContext(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(group),
		LogStreamNamePrefix: aws.String(stream),
	})

	if err != nil {
		return ErrCode("failed to describe log stream", err)
	}

	var logstream *cloudwatchlogs.LogStream
	for _, ls := range lsOut.LogStreams {
		if aws.StringValue(ls.LogStreamName) == stream {
			logstream = ls
		}
	}

	if logstream == nil {
		return apierror.New(apierror.ErrBadRequest, "logstream doesn't exist", nil)
	}

	logEvents := make([]*cloudwatchlogs.InputLogEvent, len(events))
	for i, e := range events {
		logEvents[i] = &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(e.Message),
			Timestamp: aws.Int64(e.Timestamp),
		}
	}

	out, err := c.Service.PutLogEventsWithContext(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
		SequenceToken: logstream.UploadSequenceToken,
		LogEvents:     logEvents,
	})
	if err != nil {
		return ErrCode("failed to put log events", err)
	}

	log.Debugf("output for put log events: %+v", out)

	return nil
}
