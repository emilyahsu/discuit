package core

import "github.com/discuitnet/discuit/config"

// GetDefaultImageStore returns the store name to use for saving images.
// Returns "s3" if S3 is configured, otherwise returns "disk".
func GetDefaultImageStore(cfg *config.Config) string {
	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" && cfg.S3Region != "" && cfg.S3Bucket != "" {
		return "s3"
	}
	return "disk"
} 