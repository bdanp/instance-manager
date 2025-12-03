//go:build integration
// +build integration

package test

import (
	"os"
	"testing"

	"instance-manager/pkg/aws"
	"instance-manager/pkg/config"
)

// TestAWSProviderIntegration tests the AWS provider with real AWS services
// This test requires valid AWS credentials to be set in environment variables
func TestAWSProviderIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Skip("Skipping integration test: AWS credentials not found")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create AWS provider
	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	// Test credential validation
	err = provider.ValidateCredentials()
	if err != nil {
		t.Fatalf("Invalid AWS credentials: %v", err)
	}

	// Test listing instances (should not fail even if empty)
	instances, err := provider.ListInstances()
	if err != nil {
		t.Fatalf("Failed to list instances: %v", err)
	}

	t.Logf("Found %d managed instances", len(instances))

	// Note: We don't create actual instances in this test to avoid charges
	// and cleanup complexity. The credential validation and list operation
	// are sufficient to verify the integration works.
}

// TestInstanceLifecycle would test the full instance lifecycle
// but is commented out to avoid AWS charges during testing
/*
func TestInstanceLifecycle(t *testing.T) {
	// Create a test public key file
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_key.pub")
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vwk... test@example.com"

	err := ioutil.WriteFile(keyPath, []byte(publicKey), 0644)
	if err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	// Create instance
	instanceConfig := models.InstanceConfig{
		InstanceType:     "t2.nano",
		Duration:         5 * time.Minute, // Short duration for testing
		PublicKeyPath:    keyPath,
		AvailabilityZone: "us-east-1a",
		Region:           cfg.AWS.Region,
	}

	instance, err := provider.CreateInstance(instanceConfig)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	t.Logf("Created instance: %s", instance.ID)

	// Ensure cleanup
	defer func() {
		err := provider.TerminateInstance(instance.ID)
		if err != nil {
			t.Errorf("Failed to cleanup instance %s: %v", instance.ID, err)
		}
	}()

	// Test getting instance status
	status, err := provider.GetInstanceStatus(instance.ID)
	if err != nil {
		t.Fatalf("Failed to get instance status: %v", err)
	}

	if status.ID != instance.ID {
		t.Errorf("Status ID mismatch: got %s, want %s", status.ID, instance.ID)
	}

	// Wait a bit and check if it's running
	time.Sleep(30 * time.Second)

	status, err = provider.GetInstanceStatus(instance.ID)
	if err != nil {
		t.Fatalf("Failed to get updated instance status: %v", err)
	}

	t.Logf("Instance state: %s", status.State)

	// Terminate the instance
	err = provider.TerminateInstance(instance.ID)
	if err != nil {
		t.Fatalf("Failed to terminate instance: %v", err)
	}

	t.Logf("Instance %s terminated", instance.ID)
}
*/
