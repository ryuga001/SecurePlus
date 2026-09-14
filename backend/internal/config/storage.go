package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Storage struct {
	Endpoint       string
	Region         string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	LogoPresignTTL time.Duration
}

func loadStorage() Storage {
	useSSL, _ := strconv.ParseBool(os.Getenv("S3_USE_SSL"))

	presignTTL, _ := time.ParseDuration(os.Getenv("S3_LOGO_PRESIGN_TTL"))
	if presignTTL <= IdentityTTL {
		presignTTL = 24 * time.Hour
	}

	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	return Storage{
		Endpoint:       os.Getenv("S3_ENDPOINT"),
		Region:         region,
		AccessKey:      os.Getenv("S3_ACCESS_KEY"),
		SecretKey:      os.Getenv("S3_SECRET_KEY"),
		Bucket:         os.Getenv("S3_BUCKET"),
		UseSSL:         useSSL,
		LogoPresignTTL: presignTTL,
	}
}

func (s Storage) endpointURL() string {
	endpoint := strings.TrimSuffix(strings.TrimSpace(s.Endpoint), "/")

	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}

	if s.UseSSL {
		return "https://" + endpoint
	}

	return "http://" + endpoint
}

func (s Storage) Connect(ctx context.Context) (*s3.Client, error) {
	if s.Endpoint == "" {
		return nil, errors.New("S3_ENDPOINT is not set")
	}
	if s.Bucket == "" {
		return nil, errors.New("S3_BUCKET is not set")
	}

	client := s3.New(s3.Options{
		Region:       s.Region,
		BaseEndpoint: aws.String(s.endpointURL()),
		UsePathStyle: true,
		Credentials: credentials.NewStaticCredentialsProvider(
			s.AccessKey, s.SecretKey, "",
		),
	})

	if err := s.ensureBucket(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}

func (s Storage) ensureBucket(ctx context.Context, client *s3.Client) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.Bucket)})
	if err == nil {
		return nil
	}

	if !bucketMissing(err) {
		return err
	}

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.Bucket)})
	if err != nil && !bucketExists(err) {
		return fmt.Errorf("bucket %q does not exist and could not be created: %w", s.Bucket, err)
	}

	return nil
}

func bucketMissing(err error) bool {
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}

	var noSuchBucket *types.NoSuchBucket
	if errors.As(err, &noSuchBucket) {
		return true
	}

	var response *awshttp.ResponseError
	if errors.As(err, &response) {
		return response.HTTPStatusCode() == http.StatusNotFound
	}

	return false
}

func bucketExists(err error) bool {
	var owned *types.BucketAlreadyOwnedByYou
	if errors.As(err, &owned) {
		return true
	}

	var exists *types.BucketAlreadyExists
	if errors.As(err, &exists) {
		return true
	}

	return false
}
