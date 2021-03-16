package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/google/uuid"
)

var testTime = time.Now().UTC().Truncate(time.Second)

// mockS3Client is a fake S3 client
type mockS3Client struct {
	s3iface.S3API
	t   *testing.T
	err error
}

var testJobs = map[string]Job{
	"a6d1b5a6-3a76-4d52-8856-b752afea563a": {
		Account:     "metal",
		ID:          "a6d1b5a6-3a76-4d52-8856-b752afea563a",
		Description: "first studio album",
		Details: map[string]string{
			"1":  "hit the lights",
			"2":  "the four horsement",
			"3":  "motorbreath",
			"4":  "jump in the fire",
			"5":  "pulling teeth",
			"6":  "whiplash",
			"7":  "phantom lord",
			"8":  "no remorse",
			"9":  "seek & destroy",
			"10": "metal militia",
		},
		Enabled:            true,
		ModifiedAt:         &testTime,
		ModifiedBy:         "hetfield",
		Name:               "kill 'em all",
		Group:              "metallica",
		ScheduleExpression: "00 00 25 07 *",
	},
	"55ac40d3-a902-4c70-a5b7-3e4a8679e315": {
		Account:     "metal",
		ID:          "55ac40d3-a902-4c70-a5b7-3e4a8679e315",
		Description: "second studio album",
		Details: map[string]string{
			"1": "fight fire with fire",
			"2": "ride the lightning",
			"3": "for whom the bell tolls",
			"4": "fade to black",
			"5": "trapped Under Ice",
			"6": "escape",
			"7": "creeping death",
			"8": "the call of ktulu",
		},
		Enabled:            true,
		ModifiedAt:         &testTime,
		ModifiedBy:         "ulrich",
		Name:               "ride the lilghtning",
		Group:              "metallica",
		ScheduleExpression: "00 00 27 07 *",
	},
	"f2e4ad2f-b130-4d48-83a1-2d8e842e6eec": {
		Account:     "metal",
		ID:          "f2e4ad2f-b130-4d48-83a1-2d8e842e6eec",
		Description: "third studio album",
		Details: map[string]string{
			"1": "battery",
			"2": "master of puppets",
			"3": "the thing that should not be",
			"4": "welcome home",
			"5": "disposable heroes",
			"6": "leper messiah",
			"7": "orion lord",
			"8": "damage, inc.",
		},
		Enabled:            true,
		ModifiedAt:         &testTime,
		ModifiedBy:         "burton",
		Name:               "master of puppets",
		Group:              "metallica",
		ScheduleExpression: "00 00 03 03 *",
	},
	"30b83d8a-163d-429a-86d8-beb34c266078": {
		Account:     "metal",
		ID:          "30b83d8a-163d-429a-86d8-beb34c266078",
		Description: "fourth studio album",
		Details: map[string]string{
			"1": "blackened",
			"2": "...and justice for all",
			"3": "eye of the beholder",
			"4": "one",
			"5": "the shortest straw",
			"6": "harvester of sorrow",
			"7": "the frayed ends of sanity",
			"8": "to live is to die",
			"9": "dyers eve",
		},
		Enabled:            true,
		ModifiedAt:         &testTime,
		ModifiedBy:         "newsted",
		Name:               "...and justice for all",
		Group:              "metallica",
		ScheduleExpression: "00 00 25 08 *",
	},
}

func newMockS3Client(t *testing.T, err error) s3iface.S3API {
	return &mockS3Client{
		t:   t,
		err: err,
	}
}

func (m *mockS3Client) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.t.Logf("PutObjectWithContext: %+v", input)

	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObjectWithContext(ctx aws.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.t.Logf("DeleteObjectWithContext: %+v", input)

	if strings.HasSuffix(aws.StringValue(input.Key), "/bad") {
		return nil, awserr.New(s3.ErrCodeNoSuchKey, "missing key", nil)
	}

	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObjectsWithContext(ctx aws.Context, input *s3.DeleteObjectsInput, opts ...request.Option) (*s3.DeleteObjectsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.t.Logf("DeleteObjectsWithContext: %+v", input)

	return &s3.DeleteObjectsOutput{}, nil
}

func (m *mockS3Client) GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.t.Logf("GetObjectWithContext: %+v", input)

	for k, v := range testJobs {
		if strings.HasSuffix(aws.StringValue(input.Key), k) {
			out, err := json.Marshal(v)
			if err != nil {
				return nil, awserr.New("Internal Server Error", "failed marshalling json", err)
			}
			return &s3.GetObjectOutput{Body: ioutil.NopCloser(bytes.NewReader(out))}, nil
		}
	}

	return nil, awserr.New(s3.ErrCodeNoSuchKey, aws.StringValue(input.Key)+" not found", nil)
}

func (m *mockS3Client) ListObjectsV2WithContext(ctx aws.Context, input *s3.ListObjectsV2Input, opts ...request.Option) (*s3.ListObjectsV2Output, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.t.Logf("ListObjectsV2WithContext: %+v", input)

	if aws.StringValue(input.Prefix) == "/test/group/" {
		contents := []*s3.Object{}
		for k := range testJobs {
			key := aws.StringValue(input.Prefix) + k
			obj := &s3.Object{
				Key: aws.String(key),
			}
			contents = append(contents, obj)
		}
		return &s3.ListObjectsV2Output{Contents: contents}, nil
	}

	return nil, awserr.New(s3.ErrCodeNoSuchKey, aws.StringValue(input.Prefix)+" not found", nil)
}

func TestWithStaticCredentials(t *testing.T) {
	t.Log("TODO")
}

func TestWithRegion(t *testing.T) {
	t.Log("TODO")
}

func TestWithEndpoint(t *testing.T) {
	t.Log("TODO")
}

func TestWithBucket(t *testing.T) {
	t.Log("TODO")
}

func TestWithPrefix(t *testing.T) {
	t.Log("TODO")
}

func TestCreate(t *testing.T) {
	s := S3Repository{
		S3: newMockS3Client(t, nil),
	}

	type createTest struct {
		account, group string
		job            *Job
		err            error
	}

	tests := []createTest{
		{
			job:     nil,
			account: "test",
			group:   "foo",
			err:     errors.New("derp"),
		},
		{
			account: "test",
			group:   "foo",
			job: &Job{
				ScheduleExpression: "@hourly",
			},
			err: nil,
		},
	}
	for _, v := range testJobs {
		tests = append(tests, createTest{
			account: v.Account,
			group:   v.Group,
			job:     &v,
			err:     nil,
		})
	}

	for _, tst := range tests {
		input := tst.job
		inputAccout := tst.account
		inputGroup := tst.group

		var expected *Job
		if input == nil {
			expected = nil
		} else {
			expected = &Job{
				Account:            input.Account,
				Description:        input.Description,
				Details:            input.Details,
				Enabled:            input.Enabled,
				ID:                 input.ID,
				ModifiedAt:         input.ModifiedAt,
				ModifiedBy:         input.ModifiedBy,
				Name:               input.Name,
				Group:              input.Group,
				ScheduleExpression: input.ScheduleExpression,
			}
		}

		j, err := s.Create(context.TODO(), inputAccout, inputGroup, input)
		if tst.err == nil && err != nil {
			t.Errorf("expected nil error, got %s", err)
		} else if tst.err != nil && err == nil {
			t.Errorf("expected error '%s', got nil", tst.err)
		}

		if j == nil && expected == nil {
			continue
		} else if j == nil && expected != nil {
			t.Errorf("expected %+v, got nil", expected)
			continue
		}

		// override modified at
		if j.ModifiedAt == nil {
			t.Error("expected modified at to be set, got nil")
		}
		expected.ModifiedAt = j.ModifiedAt

		// verify id is a uuid and then override it
		id, err := uuid.Parse(j.ID)
		if err != nil {
			t.Errorf("expected valid uuid as id: %s", err)
		}

		if id.String() == "" {
			t.Error("expected id to be a uuid")
		}
		expected.ID = id.String()

		if !reflect.DeepEqual(expected, j) {
			t.Errorf("expected %+v, got %+v", expected, j)
		}
	}

}

func TestDelete(t *testing.T) {
	s := S3Repository{
		S3:     newMockS3Client(t, nil),
		Bucket: "binge",
	}

	type deleteTest struct {
		account string
		group   string
		id      string
		err     error
	}

	testJobs := []deleteTest{
		// unknown account, good group, no id
		{
			account: "unknown",
			id:      "",
			group:   "group",
			err:     errors.New("derp"),
		},
		// good account, unknown group, no id
		{
			account: "test",
			id:      "",
			group:   "unknown",
			err:     errors.New("derp"),
		},
		// good account, good group, no id
		{
			account: "test",
			id:      "",
			group:   "group",
			err:     nil,
		},
		// bad id
		{
			account: "test",
			id:      "bad",
			group:   "group",
			err:     errors.New("derp"),
		},
		// good id
		{
			account: "test",
			id:      "some-id",
			group:   "group",
			err:     nil,
		},
	}

	for _, tst := range testJobs {
		t.Logf("testing delete with %+v", tst)

		err := s.Delete(context.TODO(), tst.account, tst.group, tst.id)
		t.Log("got err: ", err)
		if tst.err == nil && err != nil {
			t.Errorf("expected nil error, got %s", err)
		} else if tst.err != nil && err == nil {
			t.Errorf("expected error '%s', got nil", tst.err)
		}

	}
}

func TestUpdate(t *testing.T) {
	s := S3Repository{
		S3: newMockS3Client(t, nil),
	}

	type updateTest struct {
		id, group string
		job       *Job
		err       error
	}

	tests := []updateTest{
		{
			job:   nil,
			id:    "foo",
			group: "foo",
			err:   errors.New("derp"),
		},
	}
	for _, v := range testJobs {
		tests = append(tests, updateTest{
			job:   &v,
			id:    v.ID,
			group: v.Group,
			err:   nil,
		})
	}

	for _, tst := range tests {
		input := tst.job
		inputId := tst.id
		inputGroup := tst.group

		var expected *Job
		if input == nil {
			expected = nil
		} else {
			inputId = input.ID
			inputGroup = input.Group
			expected = &Job{
				Account:            input.Account,
				Description:        input.Description,
				Details:            input.Details,
				Enabled:            input.Enabled,
				ID:                 inputId,
				ModifiedAt:         input.ModifiedAt,
				ModifiedBy:         input.ModifiedBy,
				Name:               input.Name,
				Group:              input.Group,
				ScheduleExpression: input.ScheduleExpression,
			}
		}

		t.Log("testing with input: ", input)

		j, err := s.Update(context.TODO(), "test", inputGroup, inputId, input)
		if tst.err == nil && err != nil {
			t.Errorf("expected nil error, got %s", err)
		} else if tst.err != nil && err == nil {
			t.Errorf("expected error '%s', got nil", tst.err)
		} else if tst.err != nil && err != nil {
			continue
		}

		if j == nil && expected == nil {
			continue
		} else if j == nil && expected != nil {
			t.Errorf("expected %+v, got nil", expected)
			continue
		}

		// override modified at
		if j.ModifiedAt == nil {
			t.Error("expected modified at to be set, got nil")
		}
		expected.ModifiedAt = j.ModifiedAt

		// verify id is a uuid and then override it
		id, err := uuid.Parse(j.ID)
		if err != nil {
			t.Errorf("expected valid uuid as id: %s", err)
		}

		if id.String() == "" {
			t.Error("expected id to be a uuid")
		}
		expected.ID = id.String()

		if !reflect.DeepEqual(expected, j) {
			t.Errorf("expected %+v, got %+v", expected, j)
		}
	}

}

func TestGet(t *testing.T) {
	s := S3Repository{
		S3: newMockS3Client(t, nil),
	}

	for k, v := range testJobs {
		expected := &Job{
			Account:            v.Account,
			Description:        v.Description,
			Details:            v.Details,
			Enabled:            v.Enabled,
			ID:                 v.ID,
			ModifiedAt:         v.ModifiedAt,
			ModifiedBy:         v.ModifiedBy,
			Name:               v.Name,
			Group:              v.Group,
			ScheduleExpression: v.ScheduleExpression,
		}

		out, err := s.Get(context.TODO(), "test", "foo", k)
		if err != nil {
			t.Errorf("expected nil error, got %s", err)
		}

		if !expected.ModifiedAt.Equal(*out.ModifiedAt) {
			t.Errorf("expected modified at to be %s, got %s", expected.ModifiedAt, out.ModifiedAt)
		}
		// time has a "magic" local timezone when using time.Now() vs Parse. times need
		// to be compared with time.Equal https://github.com/golang/go/issues/17506
		out.ModifiedAt = expected.ModifiedAt

		if !reflect.DeepEqual(expected, out) {
			t.Errorf("expected %+v, got %+v", expected, out)
		}
	}

	if _, err := s.Get(context.TODO(), "test", "foo", "some-other-job"); err == nil {
		t.Error("expected error for missing key, got nil")
	}

	if _, err := s.Get(context.TODO(), "", "foo", "some-other-job"); err == nil {
		t.Error("expected error for empty account, got nil")
	}

	if _, err := s.Get(context.TODO(), "test", "", "some-other-job"); err == nil {
		t.Error("expected error for empty group, got nil")
	}

	if _, err := s.Get(context.TODO(), "test", "foo", ""); err == nil {
		t.Error("expected error for id account, got nil")
	}
}

func TestList(t *testing.T) {
	s := S3Repository{
		S3: newMockS3Client(t, nil),
	}

	expected := make([]string, 0, len(testJobs))
	for k := range testJobs {
		expected = append(expected, k)
	}

	out, err := s.List(context.TODO(), "test", "group")
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	for _, j := range expected {
		exists := false
		for _, o := range out {
			if j == o {
				exists = true
			}
		}

		if !exists {
			t.Errorf("expected %+v, got %+v", expected, out)
		}
	}

	_, err = s.List(context.TODO(), "foo", "group")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
