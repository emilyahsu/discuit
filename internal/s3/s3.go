package s3

// Config defines the configuration needed for S3 storage
type Config interface {
	GetS3Enabled() bool
	GetS3Region() string
	GetS3Bucket() string
	GetS3AccessKey() string
	GetS3SecretKey() string
	GetS3Endpoint() string
	GetS3PathPrefix() string
} 