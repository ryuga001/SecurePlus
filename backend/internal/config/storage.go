package config

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	return Storage{
		Endpoint:       os.Getenv("S3_ENDPOINT"),
		Region:         os.Getenv("S3_REGION"),
		AccessKey:      os.Getenv("S3_ACCESS_KEY"),
		SecretKey:      os.Getenv("S3_SECRET_KEY"),
		Bucket:         os.Getenv("S3_BUCKET"),
		UseSSL:         useSSL,
		LogoPresignTTL: presignTTL,
	}
}

func (s Storage) Connect(ctx context.Context) (*minio.Client, error) {
	client, err := minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
		Secure: s.UseSSL,
		Region: s.Region,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, s.Bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		if err := client.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: s.Region}); err != nil {
			return nil, err
		}
	}

	return client, nil
}
