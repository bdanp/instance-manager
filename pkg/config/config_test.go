package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"instance-manager/pkg/config"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment variables
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	originalRegion := os.Getenv("AWS_REGION")

	// Restore environment variables after test
	defer func() {
		os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
		os.Setenv("AWS_REGION", originalRegion)
	}()

	tests := []struct {
		name      string
		accessKey string
		secretKey string
		region    string
		hasError  bool
	}{
		{
			name:      "valid configuration",
			accessKey: "test-access-key",
			secretKey: "test-secret-key",
			region:    "us-west-2",
			hasError:  false,
		},
		{
			name:      "missing access key",
			accessKey: "",
			secretKey: "test-secret-key",
			region:    "us-west-2",
			hasError:  true,
		},
		{
			name:      "missing secret key",
			accessKey: "test-access-key",
			secretKey: "",
			region:    "us-west-2",
			hasError:  true,
		},
		{
			name:      "default region",
			accessKey: "test-access-key",
			secretKey: "test-secret-key",
			region:    "",
			hasError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("AWS_ACCESS_KEY_ID", tt.accessKey)
			os.Setenv("AWS_SECRET_ACCESS_KEY", tt.secretKey)
			os.Setenv("AWS_REGION", tt.region)

			cfg, err := config.LoadConfig()

			if tt.hasError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("Config is nil")
					return
				}

				if cfg.AWS.AccessKey != tt.accessKey {
					t.Errorf("AccessKey mismatch: got %s, want %s", cfg.AWS.AccessKey, tt.accessKey)
				}
				if cfg.AWS.SecretKey != tt.secretKey {
					t.Errorf("SecretKey mismatch: got %s, want %s", cfg.AWS.SecretKey, tt.secretKey)
				}

				expectedRegion := tt.region
				if expectedRegion == "" {
					expectedRegion = "us-east-1" // Default region
				}
				if cfg.AWS.Region != expectedRegion {
					t.Errorf("Region mismatch: got %s, want %s", cfg.AWS.Region, expectedRegion)
				}
			}
		})
	}
}

func TestValidatePublicKeyPath(t *testing.T) {
	// Create temporary directory and file for testing
	tempDir := t.TempDir()
	validKeyPath := filepath.Join(tempDir, "test_key.pub")
	dirPath := filepath.Join(tempDir, "test_dir")

	// Create a valid key file
	if err := os.WriteFile(validKeyPath, []byte("ssh-rsa AAAAB3NzaC1yc2E..."), 0644); err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	// Create a directory
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		hasError bool
	}{
		{
			name:     "valid key file",
			path:     validKeyPath,
			hasError: false,
		},
		{
			name:     "empty path",
			path:     "",
			hasError: true,
		},
		{
			name:     "non-existent file",
			path:     filepath.Join(tempDir, "nonexistent.pub"),
			hasError: true,
		},
		{
			name:     "directory instead of file",
			path:     dirPath,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidatePublicKeyPath(tt.path)
			if tt.hasError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
