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
func (s *S3Repository) Create(ctx context.Context, account string, job *Job) (*Job, error) {
	if job == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	// generate a new random ID for the job
	job.ID = NewID()

	return s.Update(ctx, account, job.ID, job)
}

// Delete deletes a job in the s3 jobs repository
func (s *S3Repository) Delete(ctx context.Context, account, id string) error {
	if id == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	log.Infof("deleting s3 job %+v", id)

	key := s.Prefix + "/" + account
	if !strings.HasSuffix(account, "/") && !strings.HasPrefix(id, "/") {
		key = key + "/"
	}
	key = key + id

	_, err := s.S3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return ErrCode("failed to delete object "+id, err)
	}

	return nil
}

// Get gets a job from the s3 jobs repository
func (s *S3Repository) Get(ctx context.Context, account, id string) (*Job, error) {
	if id == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	log.Infof("getting job %s", id)

	key := s.Prefix + "/" + account
	if !strings.HasSuffix(account, "/") && !strings.HasPrefix(id, "/") {
		key = key + "/"
	}
	key = key + id

	out, err := s.S3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, ErrCode("failed to get job object from s3 "+id, err)
	}

	job := &Job{}
	err = json.NewDecoder(out.Body).Decode(job)
	if err != nil {
		return nil, apierror.New(apierror.ErrBadRequest, "failed to decode json from s3", err)
	}

	log.Debugf("output from getting s3 job '%s': %+v", id, job)

	return job, nil
}

// List lists the jobs in the s3 jobs repository
func (s *S3Repository) List(ctx context.Context, account string) ([]string, error) {
	log.Infof("listing jobs for account %s", account)

	prefix := s.Prefix + "/" + account

	jobs := []string{}
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
			jId := strings.TrimPrefix(aws.StringValue(object.Key), prefix)
			jobs = append(jobs, strings.TrimPrefix(jId, "/"))
		}

		truncated = aws.BoolValue(output.IsTruncated)
		input.ContinuationToken = output.NextContinuationToken
	}

	return jobs, nil
}

// Update updates a job in the s3 jobs repository
func (s *S3Repository) Update(ctx context.Context, account, id string, job *Job) (*Job, error) {
	if job == nil || id == "" || job.ID != id {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", errors.New("empty input"))
	}

	// set the modified at to right now
	now := time.Now().UTC().Truncate(time.Second)
	job.ModifiedAt = &now

	log.Infof("updating job %+v", job)

	key := s.Prefix + "/" + account
	if !strings.HasSuffix(account, "/") && !strings.HasPrefix(id, "/") {
		key = key + "/"
	}
	key = key + id

	j, err := json.MarshalIndent(job, "", "\t")
	if err != nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", err)
	}

	out, err := s.S3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(j),
	})
	if err != nil {
		return nil, ErrCode("failed to put s3 object", err)
	}

	log.Debugf("output from s3 job put: %+v", out)

	return job, nil
}
