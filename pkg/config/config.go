package config

import (
	"errors"
	"os"
	"time"
)

// Config holds the application configuration
type Config struct {
	AWS           AWSConfig
	DefaultValues DefaultValues
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	AccessKey string
	SecretKey string
	Region    string
}

// DefaultValues holds default configuration values
type DefaultValues struct {
	InstanceType     string
	Duration         time.Duration
	AvailabilityZone string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		AWS: AWSConfig{
			AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			Region:    getEnvOrDefault("AWS_REGION", "us-east-1"),
		},
		DefaultValues: DefaultValues{
			InstanceType:     "t2.nano",
			Duration:         1 * time.Hour,
			AvailabilityZone: "us-east-1a",
		},
	}

	// Validate required environment variables
	if config.AWS.AccessKey == "" {
		return nil, errors.New("AWS_ACCESS_KEY_ID environment variable is required")
	}
	if config.AWS.SecretKey == "" {
		return nil, errors.New("AWS_SECRET_ACCESS_KEY environment variable is required")
	}

	return config, nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ValidatePublicKeyPath validates that the public key file exists and is readable
func ValidatePublicKeyPath(path string) error {
	if path == "" {
		return errors.New("public key path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("public key file does not exist")
		}
		return err
	}

	if info.IsDir() {
		return errors.New("public key path is a directory, not a file")
	}

	// Check if file is readable
	file, err := os.Open(path)
	if err != nil {
		return errors.New("cannot read public key file")
	}
	file.Close()

	return nil
}
