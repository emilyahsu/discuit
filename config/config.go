package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/discuitnet/discuit/core"
	"gopkg.in/yaml.v2"
)

// Config holds all site-wide configuration.
type Config struct {
	IsDevelopment  bool `yaml:"isDevelopment"`
	UseHTTPCookies bool `yaml:"useHttpCookies"`

	Addr    string `yaml:"addr"`
	UIProxy string `yaml:"uiProxy"`

	SiteName        string `yaml:"siteName"`
	SiteDescription string `yaml:"siteDescription"` // Used for meta tags.

	// Primary DB credentials.
	DBAddr     string `yaml:"dbAddr"`
	DBUser     string `yaml:"dbUser"`
	DBPassword string `yaml:"dbPassword"`
	DBName     string `yaml:"dbName"`

	SessionCookieName string `yaml:"sessionCookieName"`

	RedisAddress string `yaml:"redisAddress"`

	HMACSecret string `yaml:"hmacSecret"`

	CSRFOff bool `yaml:"csrfOff"`

	NoLogToFile bool `yaml:"noLogToFile"`

	PaginationLimit    int           `yaml:"paginationLimit"`
	PaginationLimitMax int           `yaml:"paginationLimitMax"`
	DefaultFeedSort    core.FeedSort `yaml:"defaultFeedSort"`

	// Captcha verification is skipped if empty.
	CaptchaSecret string `yaml:"captchaSecret"`

	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`

	DisableRateLimits bool `yaml:"disableRateLimits"`
	MaxImageSize      int  `yaml:"maxImageSize"`

	// If API requests have a URL query parameter of the form 'adminKey=value',
	// where value is AdminAPIKey, rate limits are disabled.
	AdminAPIKey string `yaml:"adminAPIKey"`

	DisableImagePosts bool `yaml:"disableImagePosts"`

	DisableForumCreation   bool `yaml:"disableForumCreation"`   // If true, only admins can create communities.
	ForumCreationReqPoints int  `yaml:"forumCreationReqPoints"` // Minimum points required for non-admins to create community, Required non-empty config field.
	MaxForumsPerUser       int  `yaml:"maxForumsPerUser"`       // Max forums one user can moderate, Required non-empty config field.

	// The location where images are saved on disk.
	ImagesFolderPath string `yaml:"imagesFolderPath"`

	// S3 configuration
	S3Enabled      bool   `yaml:"s3Enabled"`
	S3Region       string `yaml:"s3Region"`
	S3Bucket       string `yaml:"s3Bucket"`
	S3AccessKey    string `yaml:"s3AccessKey"`
	S3SecretKey    string `yaml:"s3SecretKey"`
	S3Endpoint     string `yaml:"s3Endpoint"` // Optional custom endpoint for S3-compatible services
	S3PathPrefix   string `yaml:"s3PathPrefix"` // Optional prefix for all S3 paths

	MaxImagesPerPost int `yaml:"maxImagesPerPost"`

	// For the front-end:
	CaptchaSiteKey string `yaml:"captchaSiteKey"`
	EmailContact   string `yaml:"emailContact"`
	FacebookURL    string `yaml:"facebookURL"`
	TwitterURL     string `yaml:"twitterURL"`
	InstagramURL   string `yaml:"instagramURL"`
	DiscordURL     string `yaml:"discordURL"`
	GithubURL      string `yaml:"githubURL"`
	SubstackURL    string `yaml:"substackURL"`

	WelcomeCommunity string `yaml:"welcomeCommunity"`
}

// Parse parses the yaml file at path and returns a Config.
func Parse(path string) (*Config, error) {
	c := &Config{
		// Default values.
		Addr:               ":8080",
		DBUser:             "discuit",
		SessionCookieName:  "SID",
		RedisAddress:       ":6379",
		PaginationLimit:    10,
		PaginationLimitMax: 50,
		DefaultFeedSort:    core.FeedSortHot,
		MaxImageSize:       25 * (1 << 20),
		MaxImagesPerPost:   10,

		// Required fields:
		ForumCreationReqPoints: -1,
		MaxForumsPerUser:       -1,
	}

	// Check for Heroku PORT environment variable
	if port := os.Getenv("PORT"); port != "" {
		c.Addr = ":" + port
	}

	var envConfigMap = map[string]interface{}{
		"DISCUIT_IS_DEVELOPMENT": &c.IsDevelopment,

		"DISCUIT_ADDR":     &c.Addr,
		"DISCUIT_UI_PROXY": &c.UIProxy,

		"DISCUIT_SITE_NAME":        &c.SiteName,
		"DISCUIT_SITE_DESCRIPTION": &c.SiteDescription,

		// Primary DB credentials.
		"DISCUIT_DB_ADDR":     &c.DBAddr,
		"DISCUIT_DB_USER":     &c.DBUser,
		"DISCUIT_DB_PASSWORD": &c.DBPassword,
		"DISCUIT_DB_NAME":     &c.DBName,

		"DISCUIT_SESSION_COOKIE_NAME": &c.SessionCookieName,

		"DISCUIT_REDIS_ADDRESS": &c.RedisAddress,

		"DISCUIT_HMAC_SECRET": &c.HMACSecret,

		"DISCUIT_CSRF_OFF": &c.CSRFOff,

		"DISCUIT_NO_LOG_TO_FILE": &c.NoLogToFile,

		"DISCUIT_PAGINATION_LIMIT":     &c.PaginationLimit,
		"DISCUIT_PAGINATION_LIMIT_MAX": &c.PaginationLimitMax,
		"DISCUIT_DEFAULT_FEED_SORT":    &c.DefaultFeedSort,

		// Captcha verification is skipped if empty.
		"DISCUIT_CAPTCHA_SECRET": &c.CaptchaSecret,
		"DISCUIT_CERT_FILE":      &c.CertFile,
		"DISCUIT_KEY_FILE":       &c.KeyFile,

		"DISCUIT_DISABLE_RATE_LIMITS": &c.DisableRateLimits,
		"DISCUIT_MAX_IMAGE_SIZE":      &c.MaxImageSize,

		// If API requests have a URL query parameter of the form 'adminKey=value',
		// where value is AdminApiKey, rate limits are disabled.
		"DISCUIT_ADMIN_API_KEY": &c.AdminAPIKey,

		"DISCUIT_DISABLE_IMAGE_POSTS": &c.DisableImagePosts,

		"DISCUIT_DISABLE_FORUM_CREATION":    &c.DisableForumCreation,
		"DISCUIT_FORUM_CREATION_REQ_POINTS": &c.ForumCreationReqPoints,
		"DISCUIT_MAX_FORUMS_PER_USER":       &c.MaxForumsPerUser,

		// The location where images are saved on disk.
		"DISCUIT_IMAGES_FOLDER_PATH": &c.ImagesFolderPath,

		// S3 configuration
		"DISCUIT_S3_ENABLED":    &c.S3Enabled,
		"DISCUIT_S3_REGION":     &c.S3Region,
		"DISCUIT_S3_BUCKET":     &c.S3Bucket,
		"DISCUIT_S3_ACCESS_KEY": &c.S3AccessKey,
		"DISCUIT_S3_SECRET_KEY": &c.S3SecretKey,
		"DISCUIT_S3_ENDPOINT":   &c.S3Endpoint,
		"DISCUIT_S3_PATH_PREFIX": &c.S3PathPrefix,

		// For the front-end:
		"DISCUIT_CAPTCHA_SITEKEY": &c.CaptchaSiteKey,
		"DISCUIT_EMAIL_CONTACT":   &c.EmailContact,
		"DISCUIT_FACEBOOK_URL":    &c.FacebookURL,
		"DISCUIT_TWITTER_URL":     &c.TwitterURL,
		"DISCUIT_INSTAGRAM_URL":   &c.InstagramURL,
		"DISCUIT_DISCORD_URL":     &c.DiscordURL,
		"DISCUIT_GITHUB_URL":      &c.GithubURL,
		"DISCUIT_SUBSTACK_URL":    &c.SubstackURL,

		"DISCUIT_USE_HTTP_COOKIES": &c.UseHTTPCookies,
	}

	// Attempt to unmarshal the YAML file if it exists
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) { // If the error is not because the file doesn't exist, return the error
			return nil, err
		}
		// If the file doesn't exist, just log or ignore and proceed to use environment variables
	} else {
		// If file reading was successful, unmarshal the YAML content
		if yamlErr := yaml.Unmarshal(data, &c); yamlErr != nil {
			return nil, yamlErr
		}
	}

	// Override with environment variables if present using the map
	for envVar, configField := range envConfigMap {
		if value, ok := os.LookupEnv(envVar); ok {
			switch v := configField.(type) {
			case *string:
				*v = value
			case *int:
				if i, err := strconv.Atoi(value); err == nil {
					*v = i
				}
			case *bool:
				if b, err := strconv.ParseBool(value); err == nil {
					*v = b
				}
			case *core.FeedSort:
				if err := v.UnmarshalText([]byte(value)); err != nil {
					return nil, err
				}
			default:
				return nil, errors.New("unknown type")
			}
		}
	}

	// Validation for required fields
	if c.ForumCreationReqPoints == -1 {
		return nil, errors.New("ForumCreationReqPoints cannot be (-1)")
	}
	if c.MaxForumsPerUser == -1 {
		return nil, errors.New("MaxForumsPerUser cannot be (-1)")
	}

	return c, nil
}

// Hostname returns the hostname part of c.Addr. If there's no hostname part, it
// returns an empty string.
func (c *Config) Hostname() string {
	addr := strings.TrimSpace(c.Addr)
	if n := strings.Index(addr, ":"); n != -1 {
		return addr[:n]
	}
	return addr
}

// AddressValid reports whether addr is of the form "host:port". If host is
// missing, it might return true, but if ":port" is missing it will return
// false.
func AddressValid(addr string) bool {
	s := strings.Index(addr, ":")
	if s == -1 || s > len(addr)-2 {
		return false
	}
	_, err := strconv.Atoi(addr[s+1:])
	return err == nil
}

// GetS3Enabled returns whether S3 storage is enabled
func (c *Config) GetS3Enabled() bool {
	return c.S3Enabled
}

// GetS3Region returns the S3 region
func (c *Config) GetS3Region() string {
	return c.S3Region
}

// GetS3Bucket returns the S3 bucket name
func (c *Config) GetS3Bucket() string {
	return c.S3Bucket
}

// GetS3AccessKey returns the S3 access key
func (c *Config) GetS3AccessKey() string {
	return c.S3AccessKey
}

// GetS3SecretKey returns the S3 secret key
func (c *Config) GetS3SecretKey() string {
	return c.S3SecretKey
}

// GetS3Endpoint returns the S3 endpoint
func (c *Config) GetS3Endpoint() string {
	return c.S3Endpoint
}

// GetS3PathPrefix returns the S3 path prefix
func (c *Config) GetS3PathPrefix() string {
	return c.S3PathPrefix
}
