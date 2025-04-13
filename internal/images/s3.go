package images

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/discuitnet/discuit/internal/uid"
)

// s3Store implements the store interface for AWS S3.
type s3Store struct {
	client *s3.Client
	bucket string
	prefix string
}

// newS3Store creates a new S3 store instance.
func newS3Store(region, bucket, accessKey, secretKey, endpoint, prefix string) (*s3Store, error) {
	// Create AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return &s3Store{
		client: client,
		bucket: bucket,
		prefix: prefix,
	}, nil
}

func (s *s3Store) name() string {
	return "s3"
}

// get retrieves an image from S3.
func (s *s3Store) get(r *ImageRecord) ([]byte, error) {
	key := s.objectKey(r.ID, r.Format)
	
	result, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	// Read the entire object into memory
	data := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := result.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return data, nil
}

// save stores an image in S3.
func (s *s3Store) save(r *ImageRecord, image []byte) error {
	key := s.objectKey(r.ID, r.Format)

	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(image),
	})
	if err != nil {
		return fmt.Errorf("failed to put object to S3: %w", err)
	}

	return nil
}

// delete removes an image from S3.
func (s *s3Store) delete(r *ImageRecord) error {
	key := s.objectKey(r.ID, r.Format)

	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

// objectKey generates the S3 object key for an image.
func (s *s3Store) objectKey(id uid.ID, format ImageFormat) string {
	folder, filename := idToFolder(id)
	key := path.Join(folder, filename+format.Extension())
	if s.prefix != "" {
		key = path.Join(s.prefix, key)
	}
	return key
} 