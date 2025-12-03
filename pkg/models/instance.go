package models

import (
	"fmt"
	"time"
)

// InstanceConfig represents the configuration for creating an instance
type InstanceConfig struct {
	InstanceType     string
	Duration         time.Duration
	PublicKeyPath    string
	AvailabilityZone string
	Region           string
}

// Instance represents a cloud instance
type Instance struct {
	ID               string        `json:"id"`
	InstanceType     string        `json:"instance_type"`
	Provider         string        `json:"provider"` // Add provider field
	PublicIP         string        `json:"public_ip,omitempty"`
	PrivateIP        string        `json:"private_ip,omitempty"`
	State            string        `json:"state"`
	LaunchTime       time.Time     `json:"launch_time"`
	Duration         time.Duration `json:"duration"`
	AvailabilityZone string        `json:"availability_zone"`
	KeyName          string        `json:"key_name"`
	Username         string        `json:"username"`
	ExpiresAt        time.Time     `json:"expires_at"`
}

// InstanceStatus represents the current status of an instance
type InstanceStatus struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	PublicIP  string `json:"public_ip,omitempty"`
	PrivateIP string `json:"private_ip,omitempty"`
	Username  string `json:"username"`
	Ready     bool   `json:"ready"`
}

// IsExpired checks if the instance has exceeded its duration
func (i *Instance) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// GetConnectionString returns the SSH connection string for the instance
func (i *Instance) GetConnectionString() string {
	if i.PublicIP != "" && i.Username != "" {
		return i.Username + "@" + i.PublicIP
	}
	return ""
}

// GetSSHCommand returns a complete SSH command for the instance
func (i *Instance) GetSSHCommand() string {
	if i.PublicIP != "" && i.Username != "" {
		return fmt.Sprintf("ssh -i ~/.ssh/id_rsa %s@%s", i.Username, i.PublicIP)
	}
	return ""
}

// IsReady checks if the instance is ready for connections
func (i *Instance) IsReady() bool {
	return i.State == "running" && i.PublicIP != ""
}

// NeedsIPUpdate checks if instance needs IP information updated
func (i *Instance) NeedsIPUpdate() bool {
	return (i.State == "running" || i.State == "pending") && i.PublicIP == ""
}

// InstanceRecord represents an instance record for storage
type InstanceRecord struct {
	Instance  *Instance `json:"instance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
