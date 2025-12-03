package scheduler_test

import (
	"context"
	"testing"
	"time"

	"instance-manager/internal/scheduler"
	"instance-manager/pkg/models"
	"instance-manager/pkg/storage"

	"github.com/sirupsen/logrus"
)

// MockProvider implements the CloudProvider interface for testing
type MockProvider struct {
	instances      map[string]*models.InstanceStatus
	startCalls     []string
	stopCalls      []string
	terminateCalls []string
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		instances:      make(map[string]*models.InstanceStatus),
		startCalls:     make([]string, 0),
		stopCalls:      make([]string, 0),
		terminateCalls: make([]string, 0),
	}
}

func (m *MockProvider) CreateInstance(config models.InstanceConfig) (*models.Instance, error) {
	// Not used in scheduler tests
	return nil, nil
}

func (m *MockProvider) GetInstanceStatus(instanceID string) (*models.InstanceStatus, error) {
	if status, exists := m.instances[instanceID]; exists {
		return status, nil
	}

	// Default status for testing
	return &models.InstanceStatus{
		ID:       instanceID,
		State:    "running",
		PublicIP: "1.2.3.4",
		Username: "ec2-user",
		Ready:    true,
	}, nil
}

func (m *MockProvider) StartInstance(instanceID string) error {
	m.startCalls = append(m.startCalls, instanceID)
	if status, exists := m.instances[instanceID]; exists {
		status.State = "pending"
	}
	return nil
}

func (m *MockProvider) StopInstance(instanceID string) error {
	m.stopCalls = append(m.stopCalls, instanceID)
	if status, exists := m.instances[instanceID]; exists {
		status.State = "stopping"
	}
	return nil
}

func (m *MockProvider) TerminateInstance(instanceID string) error {
	m.terminateCalls = append(m.terminateCalls, instanceID)
	if status, exists := m.instances[instanceID]; exists {
		status.State = "terminating"
	}
	return nil
}

func (m *MockProvider) ListInstances() ([]*models.Instance, error) {
	// Not used in scheduler tests
	return []*models.Instance{}, nil
}

func (m *MockProvider) ValidateCredentials() error {
	return nil
}

func (m *MockProvider) SetInstanceStatus(instanceID, state string) {
	if _, exists := m.instances[instanceID]; !exists {
		m.instances[instanceID] = &models.InstanceStatus{
			ID:       instanceID,
			Username: "ec2-user",
		}
	}
	m.instances[instanceID].State = state
}

func TestSchedulerExpiredInstance(t *testing.T) {
	// Create mock provider and storage
	provider := NewMockProvider()
	storage := storage.NewFileStorage(t.TempDir() + "/test.json")

	// Create an expired instance
	expiredInstance := &models.Instance{
		ID:         "i-expired123",
		State:      "running",
		LaunchTime: time.Now().Add(-2 * time.Hour),
		Duration:   1 * time.Hour,
		ExpiresAt:  time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	// Save instance to storage
	err := storage.SaveInstance(expiredInstance)
	if err != nil {
		t.Fatalf("Failed to save instance: %v", err)
	}

	// Set instance as running in provider
	provider.SetInstanceStatus("i-expired123", "running")

	// Create scheduler
	sched := scheduler.NewScheduler(provider, storage)
	sched.SetLogLevel(logrus.DebugLevel)

	// Run scheduler once
	sched.RunOnce()

	// Check that stop was called
	if len(provider.stopCalls) != 1 {
		t.Errorf("Expected 1 stop call, got %d", len(provider.stopCalls))
	}
	if provider.stopCalls[0] != "i-expired123" {
		t.Errorf("Expected stop call for i-expired123, got %s", provider.stopCalls[0])
	}
}

func TestSchedulerStoppedInstanceWithExtendedTTL(t *testing.T) {
	// Create mock provider and storage
	provider := NewMockProvider()
	storage := storage.NewFileStorage(t.TempDir() + "/test.json")

	// Create a stopped instance with extended TTL
	stoppedInstance := &models.Instance{
		ID:         "i-stopped123",
		State:      "running", // State in storage (will be updated by scheduler)
		LaunchTime: time.Now().Add(-30 * time.Minute),
		Duration:   2 * time.Hour,                 // Extended duration
		ExpiresAt:  time.Now().Add(1 * time.Hour), // Extended to 1 hour in future
	}

	// Save instance to storage
	err := storage.SaveInstance(stoppedInstance)
	if err != nil {
		t.Fatalf("Failed to save instance: %v", err)
	}

	// Set instance as stopped in provider (simulating user stopped it manually)
	provider.SetInstanceStatus("i-stopped123", "stopped")

	// Create scheduler
	sched := scheduler.NewScheduler(provider, storage)
	sched.SetLogLevel(logrus.DebugLevel)

	// Run scheduler once
	sched.RunOnce()

	// Check that start was called
	if len(provider.startCalls) != 1 {
		t.Errorf("Expected 1 start call, got %d", len(provider.startCalls))
	}
	if provider.startCalls[0] != "i-stopped123" {
		t.Errorf("Expected start call for i-stopped123, got %s", provider.startCalls[0])
	}
}

func TestSchedulerStateSync(t *testing.T) {
	// Create mock provider and storage
	provider := NewMockProvider()
	storage := storage.NewFileStorage(t.TempDir() + "/test.json")

	// Create an instance
	instance := &models.Instance{
		ID:         "i-sync123",
		State:      "pending", // State in storage
		LaunchTime: time.Now(),
		Duration:   1 * time.Hour,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	// Save instance to storage
	err := storage.SaveInstance(instance)
	if err != nil {
		t.Fatalf("Failed to save instance: %v", err)
	}

	// Set instance as running in provider (different from storage)
	provider.SetInstanceStatus("i-sync123", "running")

	// Create scheduler
	sched := scheduler.NewScheduler(provider, storage)
	sched.SetLogLevel(logrus.DebugLevel)

	// Run scheduler once
	sched.RunOnce()

	// Check that state was synced in storage
	updatedInstance, err := storage.GetInstance("i-sync123")
	if err != nil {
		t.Fatalf("Failed to get updated instance: %v", err)
	}

	if updatedInstance.State != "running" {
		t.Errorf("Expected state to be synced to 'running', got %s", updatedInstance.State)
	}
}

func TestSchedulerReloadInterval(t *testing.T) {
	// Create mock provider and storage
	provider := NewMockProvider()
	storage := storage.NewFileStorage(t.TempDir() + "/test.json")

	// Create scheduler with very short intervals for testing
	sched := scheduler.NewScheduler(provider, storage)
	sched.SetLogLevel(logrus.DebugLevel)

	// Start scheduler
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sched.Start()

	// Let it run briefly
	<-ctx.Done()

	sched.Stop()

	// Test passes if no errors occur during the brief run
}
