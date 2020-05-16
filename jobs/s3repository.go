package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/YaleSpinup/minion/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"
)

// S3RepositoryOption is a function to set repository options
type S3RepositoryOption func(*S3Repository)

// S3Repository is an implementation of a jobs respository in S3
type S3Repository struct {
	S3     s3iface.S3API
	Bucket string
	Prefix string
	config *aws.Config
}

// NewDefaultRepository creates a new repository from the default config data
func NewDefaultRepository(config map[string]interface{}) (*S3Repository, error) {
	log.Debug("creating new default repository")

	var akid, secret, token, region, endpoint, bucket, prefix string
	if v, ok := config["akid"].(string); ok {
		akid = v
	}

	if v, ok := config["secret"].(string); ok {
		secret = v
	}

	if v, ok := config["token"].(string); ok {
		token = v
	}

	if v, ok := config["region"].(string); ok {
		region = v
	}

	if v, ok := config["endpoint"].(string); ok {
		endpoint = v
	}

	if v, ok := config["bucket"].(string); ok {
		bucket = v
	}

	if v, ok := config["prefix"].(string); ok {
		prefix = v
	}

	opts := []S3RepositoryOption{
		WithStaticCredentials(akid, secret, token),
	}

	if region != "" {
		opts = append(opts, WithRegion(region))
	}

	if endpoint != "" {
		opts = append(opts, WithEndpoint(endpoint))
	}

	if bucket != "" {
		opts = append(opts, WithBucket(bucket))
	}

	if prefix != "" {
		opts = append(opts, WithPrefix(prefix))
	}

	return New(opts...)
}

// New creates an S3Repository from a list of S3RepositoryOption functions
func New(opts ...S3RepositoryOption) (*S3Repository, error) {
	log.Info("creating new s3 repository provider")

	s := S3Repository{}
	s.config = aws.NewConfig()

	for _, opt := range opts {
		opt(&s)
	}

	sess := session.Must(session.NewSession(s.config))

	s.S3 = s3.New(sess)
	return &s, nil
}

// WithStaticCredentials authenticates with AWS static credentials (key, secret, token)
func WithStaticCredentials(akid, secret, token string) S3RepositoryOption {
	return func(s *S3Repository) {
		log.Debugf("setting static credentials with akid %s", akid)
		s.config.WithCredentials(credentials.NewStaticCredentials(akid, secret, token))
	}
}

// WithRegion sets the region for the S3Repository
func WithRegion(region string) S3RepositoryOption {
	return func(s *S3Repository) {
		log.Debugf("setting region %s", region)
		s.config.WithRegion(region)
	}
}

// WithEndpoint sets the endpoint for the S3Repository
func WithEndpoint(endpoint string) S3RepositoryOption {
	return func(s *S3Repository) {
		log.Debugf("setting endpoint %s", endpoint)
		s.config.WithEndpoint(endpoint)
	}
}

// WithBucket sets the bucket for the S3Repository
func WithBucket(bucket string) S3RepositoryOption {
	return func(s *S3Repository) {
		log.Debugf("setting bucket %s", bucket)
		s.Bucket = bucket
	}
}

// WithPrefix sets the bucket prefix for the S3Repository
func WithPrefix(prefix string) S3RepositoryOption {
	return func(s *S3Repository) {
		log.Debugf("setting bucket prefix %s", prefix)
		s.Prefix = prefix
	}
}

// func WithLoggingBucket(bucket string) S3RepositoryOption {
// 	return func(s *S3Repository) {
// 		s.LoggingBucket = bucket
// 	}
// }

// func WithLoggingBucketPrefix(prefix string) S3RepositoryOption {
// 	return func(s *S3Repository) {
// 		s.LoggingBucketPrefix = prefix
// 	}
// }

// Create creates a job in the s3 jobs repository
func (s *S3Repository) Create(ctx context.Context, account, group string, job *Job) (*Job, error) {
	if account == "" || group == "" || job == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	// generate a new random ID for the job
	job.ID = NewID()

	return s.Update(ctx, account, group, job.ID, job)
}

// Delete deletes a job in the s3 jobs repository
func (s *S3Repository) Delete(ctx context.Context, account, group, id string) error {
	if account == "" || group == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	log.Infof("deleting job from s3 %s/%s/%s", account, group, id)

	key := s.Prefix + "/" + account
	if !strings.HasSuffix(account, "/") && !strings.HasPrefix(group, "/") {
		key = key + "/"
	}
	key = key + group

	if id == "" {
		return s.deletePath(ctx, key)
	}

	if !strings.HasSuffix(group, "/") && !strings.HasPrefix(id, "/") {
		key = key + "/"
	}

	key = key + id
	return s.deleteObject(ctx, key)
}

func (s *S3Repository) deletePath(ctx context.Context, prefix string) error {
	log.Warnf("recursively deleting objects with prefix %s from bucket %s", prefix, s.Bucket)

	jobs, err := s.listObjects(ctx, prefix)
	if err != nil {
		return err
	}

	log.Debugf("got list of objects with prefix %s: %+v", prefix, jobs)

	// TODO: handle the case of deleting more than 1000 objects?
	if len(jobs) >= 1000 {
		return errors.New("cannot delete more than 1000 jobs at one time")
	}

	objs := make([]*s3.ObjectIdentifier, len(jobs))
	for i, obj := range jobs {
		objs[i] = &s3.ObjectIdentifier{
			Key: aws.String(prefix + "/" + obj),
		}
	}

	if _, err = s.S3.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.Bucket),
		Delete: &s3.Delete{
			Objects: objs,
		},
	}); err != nil {
		return ErrCode("failed to delete objects with prefix "+prefix, err)
	}

	return nil
}

func (s *S3Repository) deleteObject(ctx context.Context, key string) error {
	log.Warnf("deleting objects with key %s from bucket %s", key, s.Bucket)

	if _, err := s.S3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	}); err != nil {
		return ErrCode("failed to delete object "+key, err)
	}

	return nil
}

// Get gets a job from the s3 jobs repository
func (s *S3Repository) Get(ctx context.Context, account, group, id string) (*Job, error) {
	if account == "" || group == "" || id == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	log.Infof("getting job %s/%s/%s", account, group, id)

	key := s.Prefix + "/" + account
	if !strings.HasSuffix(account, "/") && !strings.HasPrefix(group, "/") {
		key = key + "/"
	}
	key = key + group

	if !strings.HasSuffix(group, "/") && !strings.HasPrefix(id, "/") {
		key = key + "/"
	}

	key = key + id

	log.Debugf("getting object (account: %s, group: %s, id: %s, key: '%s')", account, group, id, key)

	out, err := s.S3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, ErrCode("failed to get job object from s3 "+id, err)
	}
	defer out.Body.Close()

	job := &Job{}
	err = json.NewDecoder(out.Body).Decode(job)
	if err != nil {
		return nil, apierror.New(apierror.ErrBadRequest, "failed to decode json from s3", err)
	}

	log.Debugf("output from getting s3 job '%s': %+v", id, job)

	return job, nil
}

// List lists the jobs in the s3 jobs repository.  If group is empty, all jobs are returned from the
// account.  If some of those jobs are in a group, the group is prefixed with the job id in the response.
func (s *S3Repository) List(ctx context.Context, account, group string) ([]string, error) {
	if account == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	log.Infof("listing jobs for account '%s', group '%s'", account, group)

	prefix := account
	if group != "" {
		if !strings.HasSuffix(account, "/") && !strings.HasPrefix(group, "/") {
			prefix = prefix + "/"
		}
		prefix = prefix + group
	}

	log.Infof("listing jobs %s", prefix)

	prefix = s.Prefix + "/" + prefix

	return s.listObjects(ctx, prefix)
}

func (s *S3Repository) listObjects(ctx context.Context, prefix string) ([]string, error) {
	objs := []string{}

	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	input := s3.ListObjectsV2Input{
		Bucket: aws.String(s.Bucket),
		Prefix: aws.String(prefix),
	}

	truncated := true
	for truncated {
		output, err := s.S3.ListObjectsV2WithContext(ctx, &input)
		if err != nil {
			return nil, ErrCode("failed to list job objects from s3 ", err)
		}

		for _, object := range output.Contents {
			id := strings.TrimPrefix(aws.StringValue(object.Key), prefix)
			objs = append(objs, strings.TrimPrefix(id, "/"))
		}

		truncated = aws.BoolValue(output.IsTruncated)
		input.ContinuationToken = output.NextContinuationToken
	}

	return objs, nil
}

// Update updates a job in the s3 jobs repository
func (s *S3Repository) Update(ctx context.Context, account, group, id string, job *Job) (*Job, error) {
	if account == "" || group == "" || id == "" || job == nil || job.ID != id {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	// set the modified at to right now
	now := time.Now().UTC().Truncate(time.Second)
	job.ModifiedAt = &now

	log.Infof("updating job %s/%s/%s", account, group, id)

	key := s.Prefix + "/" + account
	if !strings.HasSuffix(account, "/") && !strings.HasPrefix(group, "/") {
		key = key + "/"
	}
	key = key + group

	if !strings.HasSuffix(group, "/") && !strings.HasPrefix(id, "/") {
		key = key + "/"
	}

	key = key + id

	log.Debugf("updating %s with job %+v", key, job)

	j, err := json.MarshalIndent(job, "", "\t")
	if err != nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", err)
	}

	out, err := s.S3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Body:        bytes.NewReader(j),
		Bucket:      aws.String(s.Bucket),
		ContentType: aws.String("application/json"),
		Key:         aws.String(key),
	})
	if err != nil {
		return nil, ErrCode("failed to put s3 object", err)
	}

	log.Debugf("output from s3 job put: %+v", out)

	return job, nil
}
