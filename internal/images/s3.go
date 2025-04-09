package images

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Store struct {
	client *s3.Client
	bucket string
}

func newS3Store(accessKey, secretKey, region, bucket string) (*s3Store, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := s3.NewFromConfig(cfg)
	return &s3Store{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *s3Store) name() string {
	return "s3"
}

func (s *s3Store) get(r *ImageRecord) ([]byte, error) {
	key := s.objectKey(r)
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %v", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

func (s *s3Store) save(r *ImageRecord, image []byte) error {
	key := s.objectKey(r)
	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(image),
		ContentType: aws.String("image/" + string(r.Format)),
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %v", err)
	}
	return nil
}

func (s *s3Store) delete(r *ImageRecord) error {
	key := s.objectKey(r)
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %v", err)
	}
	return nil
}

func (s *s3Store) objectKey(r *ImageRecord) string {
	folder, filename := idToFolder(r.ID)
	return path.Join(folder, filename+r.Format.Extension())
} 