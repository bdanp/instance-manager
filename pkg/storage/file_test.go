package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"instance-manager/pkg/models"
	"instance-manager/pkg/storage"
)

func TestFileStorage_SaveAndGetInstance(t *testing.T) {
	// Create temporary file for testing
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_instances.json")

	fs := storage.NewFileStorage(filePath)

	// Create test instance
	instance := &models.Instance{
		ID:               "i-123456789",
		InstanceType:     "t2.nano",
		State:            "running",
		LaunchTime:       time.Now(),
		Duration:         1 * time.Hour,
		AvailabilityZone: "us-east-1a",
		Username:         "ec2-user",
		ExpiresAt:        time.Now().Add(1 * time.Hour),
	}

	// Test save
	err := fs.SaveInstance(instance)
	if err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}

	// Test get
	retrieved, err := fs.GetInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}

	// Verify data
	if retrieved.ID != instance.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, instance.ID)
	}
	if retrieved.InstanceType != instance.InstanceType {
		t.Errorf("InstanceType mismatch: got %s, want %s", retrieved.InstanceType, instance.InstanceType)
	}
}

func TestFileStorage_ListInstances(t *testing.T) {
	// Create temporary file for testing
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_instances.json")

	fs := storage.NewFileStorage(filePath)

	// Test empty list
	instances, err := fs.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("Expected empty list, got %d instances", len(instances))
	}

	// Add test instances
	instance1 := &models.Instance{
		ID:        "i-123456789",
		State:     "running",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	instance2 := &models.Instance{
		ID:        "i-987654321",
		State:     "pending",
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}

	err = fs.SaveInstance(instance1)
	if err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}
	err = fs.SaveInstance(instance2)
	if err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}

	// Test list with instances
	instances, err = fs.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}
}

func TestFileStorage_UpdateInstance(t *testing.T) {
	// Create temporary file for testing
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_instances.json")

	fs := storage.NewFileStorage(filePath)

	// Create and save instance
	instance := &models.Instance{
		ID:        "i-123456789",
		State:     "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := fs.SaveInstance(instance)
	if err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}

	// Update instance
	instance.State = "running"
	instance.PublicIP = "1.2.3.4"

	err = fs.UpdateInstance(instance)
	if err != nil {
		t.Fatalf("UpdateInstance failed: %v", err)
	}

	// Verify update
	updated, err := fs.GetInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}

	if updated.State != "running" {
		t.Errorf("State not updated: got %s, want running", updated.State)
	}
	if updated.PublicIP != "1.2.3.4" {
		t.Errorf("PublicIP not updated: got %s, want 1.2.3.4", updated.PublicIP)
	}
}

func TestFileStorage_DeleteInstance(t *testing.T) {
	// Create temporary file for testing
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_instances.json")

	fs := storage.NewFileStorage(filePath)

	// Create and save instance
	instance := &models.Instance{
		ID:        "i-123456789",
		State:     "running",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := fs.SaveInstance(instance)
	if err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}

	// Delete instance
	err = fs.DeleteInstance(instance.ID)
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Verify deletion
	_, err = fs.GetInstance(instance.ID)
	if err == nil {
		t.Error("Expected error when getting deleted instance")
	}
}

func TestFileStorage_GetExpiredInstances(t *testing.T) {
	// Create temporary file for testing
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_instances.json")

	fs := storage.NewFileStorage(filePath)

	// Create expired and non-expired instances
	expiredInstance := &models.Instance{
		ID:        "i-expired",
		State:     "running",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	activeInstance := &models.Instance{
		ID:        "i-active",
		State:     "running",
		ExpiresAt: time.Now().Add(1 * time.Hour), // Not expired
	}

	err := fs.SaveInstance(expiredInstance)
	if err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}
	err = fs.SaveInstance(activeInstance)
	if err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}

	// Get expired instances
	expired, err := fs.GetExpiredInstances()
	if err != nil {
		t.Fatalf("GetExpiredInstances failed: %v", err)
	}

	if len(expired) != 1 {
		t.Errorf("Expected 1 expired instance, got %d", len(expired))
	}
	if len(expired) > 0 && expired[0].ID != "i-expired" {
		t.Errorf("Wrong expired instance: got %s, want i-expired", expired[0].ID)
	}
}
