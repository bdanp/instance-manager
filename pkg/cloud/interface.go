package cloud

import (
	"instance-manager/pkg/models"
)

// CloudProvider defines the interface for cloud providers
type CloudProvider interface {
	// CreateInstance creates a new instance with the given configuration
	CreateInstance(config models.InstanceConfig) (*models.Instance, error)

	// GetInstanceStatus retrieves the current status of an instance
	GetInstanceStatus(instanceID string) (*models.InstanceStatus, error)

	// StartInstance starts a stopped instance
	StartInstance(instanceID string) error

	// StopInstance stops a running instance (without terminating)
	StopInstance(instanceID string) error

	// TerminateInstance terminates the specified instance
	TerminateInstance(instanceID string) error

	// ListInstances returns a list of all instances managed by this provider
	ListInstances() ([]*models.Instance, error)

	// ValidateCredentials checks if the provider credentials are valid
	ValidateCredentials() error
}

// ProviderConfig represents configuration common to all cloud providers
type ProviderConfig struct {
	Region string
}
