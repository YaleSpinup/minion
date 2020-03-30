package cloudwatchlogs

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/minion/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/pkg/errors"
)

// mockCWLClient is a fake cloudwatchlogs client
type mockCWLClient struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	t   *testing.T
	err error
}

func newmockCWLClient(t *testing.T, err error) cloudwatchlogsiface.CloudWatchLogsAPI {
	return &mockCWLClient{
		t:   t,
		err: err,
	}
}

func (m *mockCWLClient) GetLogEventsWithContext(ctx context.Context, input *cloudwatchlogs.GetLogEventsInput, opts ...request.Option) (*cloudwatchlogs.GetLogEventsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.GetLogEventsOutput{}, nil
}

func (m *mockCWLClient) CreateLogGroupWithContext(ctx context.Context, input *cloudwatchlogs.CreateLogGroupInput, opts ...request.Option) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.CreateLogGroupOutput{}, nil
}

func (m *mockCWLClient) PutRetentionPolicyWithContext(ctx context.Context, input *cloudwatchlogs.PutRetentionPolicyInput, opts ...request.Option) (*cloudwatchlogs.PutRetentionPolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.PutRetentionPolicyOutput{}, nil
}

func (m *mockCWLClient) CreateLogStreamWithContext(ctx context.Context, input *cloudwatchlogs.CreateLogStreamInput, opts ...request.Option) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.CreateLogStreamOutput{}, nil
}

func (m *mockCWLClient) DescribeLogStreamsWithContext(ctx context.Context, input *cloudwatchlogs.DescribeLogStreamsInput, opts ...request.Option) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.DescribeLogStreamsOutput{
		LogStreams: []*cloudwatchlogs.LogStream{
			&cloudwatchlogs.LogStream{
				LogStreamName:       aws.String("foo"),
				UploadSequenceToken: aws.String("12345"),
			},
			&cloudwatchlogs.LogStream{
				LogStreamName:       aws.String("bar"),
				UploadSequenceToken: aws.String("67890"),
			},
			&cloudwatchlogs.LogStream{
				LogStreamName:       aws.String("bad"),
				UploadSequenceToken: aws.String("00000"),
			},
		},
	}, nil
}

func (m *mockCWLClient) PutLogEventsWithContext(ctx context.Context, input *cloudwatchlogs.PutLogEventsInput, opts ...request.Option) (*cloudwatchlogs.PutLogEventsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.LogStreamName) == "bad" {
		return &cloudwatchlogs.PutLogEventsOutput{}, errors.New("boom")
	}

	return &cloudwatchlogs.PutLogEventsOutput{
		NextSequenceToken: aws.String("111213"),
	}, nil
}

func TestNewSession(t *testing.T) {
	cw := NewSession("foo", "bar", "baz")
	to := reflect.TypeOf(cw).String()
	if to != "cloudwatchlogs.CloudWatchLogs" {
		t.Errorf("expected type to be 'cloudwatchlogs.CloudWatchLogs', got %s", to)
	}
}

func TestGetLogEvents(t *testing.T) {
	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}
	expected := &cloudwatchlogs.GetLogEventsOutput{}
	out, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("clu0"),
		LogStreamName: aws.String("logStream0"),
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	if _, err = client.GetLogEvents(context.TODO(), nil); err == nil {
		t.Errorf("expected err for nil input")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	_, err = client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{})
	if err == nil {
		t.Error("expected error, got nil")
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}

func TestCreateLogGroup(t *testing.T) {
	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}
	if err := client.CreateLogGroup(context.TODO(), "log-group-01", make(map[string]*string)); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.CreateLogGroup(context.TODO(), "log-group-01", map[string]*string{
		"foo": aws.String("bar"),
		"baz": aws.String("biz"),
	}); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.CreateLogGroup(context.TODO(), "", make(map[string]*string)); err == nil {
		t.Errorf("expected err for empty input")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeResourceAlreadyExistsException, "The log group already exists.", nil))}
	if err := client.CreateLogGroup(context.TODO(), "log-group-01", make(map[string]*string)); err != nil {
		t.Error("expected error, got nil")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	if err := client.CreateLogGroup(context.TODO(), "log-group-01", make(map[string]*string)); err == nil {
		t.Error("expected error, got nil")
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}

func TestUpdateRetention(t *testing.T) {
	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}
	if err := client.UpdateRetention(context.TODO(), "log-group-01", int64(365)); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.UpdateRetention(context.TODO(), "", int64(0)); err == nil {
		t.Errorf("expected err for nil input")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	if err := client.UpdateRetention(context.TODO(), "log-group-01", int64(365)); err == nil {
		t.Error("expected error, got nil")
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}

func TestCreateLogStream(t *testing.T) {
	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}
	if err := client.CreateLogStream(context.TODO(), "log-group-01", "log-stream-01"); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.CreateLogStream(context.TODO(), "", "foo"); err == nil {
		t.Errorf("expected err for empty group")
	}

	if err := client.CreateLogStream(context.TODO(), "foo", ""); err == nil {
		t.Errorf("expected err for empty stream")
	}

	if err := client.CreateLogStream(context.TODO(), "", ""); err == nil {
		t.Errorf("expected err for empty group and stream")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	if err := client.CreateLogStream(context.TODO(), "foo", "bar"); err == nil {
		t.Error("expected error, got nil")
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}

func TestLogEvents(t *testing.T) {
	testEvents := []*Event{
		&Event{
			Timestamp: int64(12345),
			Message:   "halp, broke!",
		},
		&Event{
			Timestamp: int64(67890),
			Message:   "werks",
		},
	}

	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}

	if err := client.LogEvent(context.TODO(), "", "bar", testEvents); err == nil {
		t.Error("expected error for empty group, got nil")
	}

	if err := client.LogEvent(context.TODO(), "foo", "", testEvents); err == nil {
		t.Error("expected error for empty stream, got nil")
	}

	if err := client.LogEvent(context.TODO(), "foo", "bar", []*Event{}); err == nil {
		t.Error("expected error for empty events, got nil")
	}

	if err := client.LogEvent(context.TODO(), "foo", "unknown", testEvents); err == nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.LogEvent(context.TODO(), "foo", "bad", testEvents); err == nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.LogEvent(context.TODO(), "foo", "bar", testEvents); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	if err := client.LogEvent(context.TODO(), "foo", "bar", testEvents); err == nil {
		t.Error("expected error for error from describe log group, got nil")
	}

}
