package provider

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const s3PageSize = 1000

var ErrInvalidS3Target = errors.New("target does not identify an S3 bucket")

type S3Source struct {
	client      *Client
	credentials aws.CredentialsProvider
	region      string

	mu      sync.Mutex
	buckets map[string]*s3.Client
}

func NewS3Source(
	ctx context.Context,
	client *Client,
	accessKeyID, secretAccessKey, region string,
) (*S3Source, error) {
	if err := client.AWSCallerIdentity(ctx, accessKeyID, secretAccessKey, region); err != nil {
		return nil, err
	}

	return &S3Source{
		client:      client,
		credentials: awscreds.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		region:      region,
		buckets:     map[string]*s3.Client{},
	}, nil
}

func (s *S3Source) List(ctx context.Context, target string, emit func(File) error) error {
	bucket, prefix := splitBucketPath(target)
	if bucket == "" {
		return ErrInvalidS3Target
	}

	api, err := s.bucketClient(ctx, bucket)
	if err != nil {
		return err
	}

	input := &s3.ListObjectsV2Input{Bucket: aws.String(bucket), MaxKeys: aws.Int32(s3PageSize)}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	pages := s3.NewListObjectsV2Paginator(api, input)

	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return s3Error(StageList, err)
		}

		for _, object := range page.Contents {
			key := aws.ToString(object.Key)
			size := aws.ToInt64(object.Size)

			if strings.HasSuffix(key, "/") && size == 0 {
				continue
			}

			if err := emit(File{
				Key:        bucket + "/" + key,
				Name:       key,
				Size:       size,
				ModifiedAt: aws.ToTime(object.LastModified),
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *S3Source) Open(ctx context.Context, file File) (io.ReadCloser, error) {
	bucket, key := splitBucketPath(file.Key)
	if bucket == "" || key == "" {
		return nil, ErrInvalidS3Target
	}

	api, err := s.bucketClient(ctx, bucket)
	if err != nil {
		return nil, err
	}

	output, err := api.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, s3Error(StageDownload, err)
	}

	return output.Body, nil
}

func (s *S3Source) bucketClient(ctx context.Context, bucket string) (*s3.Client, error) {
	s.mu.Lock()
	cached, ok := s.buckets[bucket]
	s.mu.Unlock()

	if ok {
		return cached, nil
	}

	region := s.region

	output, err := s.api(region, bucket).HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	switch {
	case err == nil && aws.ToString(output.BucketRegion) != "":
		region = aws.ToString(output.BucketRegion)
	case err != nil:
		redirected := redirectRegion(err)
		if redirected == "" {
			return nil, s3Error(StageList, err)
		}

		region = redirected
	}

	api := s.api(region, bucket)

	s.mu.Lock()
	s.buckets[bucket] = api
	s.mu.Unlock()

	return api, nil
}

func (s *S3Source) api(region, bucket string) *s3.Client {
	return s3.New(s3.Options{
		Region:       region,
		Credentials:  s.credentials,
		HTTPClient:   s.client.http,
		UsePathStyle: strings.Contains(bucket, "."),
	})
}

func splitBucketPath(value string) (string, string) {
	bucket, rest, _ := strings.Cut(strings.TrimPrefix(value, "/"), "/")

	return bucket, rest
}

func redirectRegion(err error) string {
	var response *awshttp.ResponseError
	if !errors.As(err, &response) || response.Response == nil {
		return ""
	}

	return response.Response.Header.Get("X-Amz-Bucket-Region")
}

func s3Error(stage string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	failure := &Error{Provider: ProviderAWS, Stage: stage, Reason: ReasonUnavailable}

	var response *awshttp.ResponseError
	if errors.As(err, &response) {
		failure.HTTPStatus = response.HTTPStatusCode()
		failure.requestID = response.RequestID
		failure.Reason = Classify(failure.HTTPStatus, nil)
	}

	var api smithy.APIError
	if errors.As(err, &api) {
		failure.code = api.ErrorCode()

		if reason := s3Reason(api.ErrorCode()); reason != "" {
			failure.Reason = reason
		}
	}

	return failure
}

func s3Reason(code string) string {
	switch code {
	case "NoSuchBucket", "NoSuchKey", "NotFound":
		return ReasonNotFound
	case "AccessDenied", "AllAccessDisabled", "AccountProblem", "Forbidden":
		return ReasonPermissionDenied
	case "InvalidAccessKeyId", "SignatureDoesNotMatch", "ExpiredToken", "InvalidToken":
		return ReasonAuthFailed
	case "SlowDown", "Throttling", "ThrottlingException", "RequestLimitExceeded", "ServiceUnavailable":
		return ReasonRateLimited
	case "InternalError":
		return ReasonUnavailable
	}

	return ""
}
