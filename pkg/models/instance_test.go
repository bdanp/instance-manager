package models_test

import (
	"testing"
	"time"

	"instance-manager/pkg/models"
)

func TestInstance_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		instance *models.Instance
		expected bool
	}{
		{
			name: "not expired",
			instance: &models.Instance{
				ID:        "i-123456789",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired",
			instance: &models.Instance{
				ID:        "i-123456789",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "just expired",
			instance: &models.Instance{
				ID:        "i-123456789",
				ExpiresAt: time.Now().Add(-1 * time.Second),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.instance.IsExpired(); got != tt.expected {
				t.Errorf("Instance.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInstance_GetConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		instance *models.Instance
		expected string
	}{
		{
			name: "with public IP and username",
			instance: &models.Instance{
				PublicIP: "1.2.3.4",
				Username: "ec2-user",
			},
			expected: "ec2-user@1.2.3.4",
		},
		{
			name: "without public IP",
			instance: &models.Instance{
				Username: "ec2-user",
			},
			expected: "",
		},
		{
			name: "without username",
			instance: &models.Instance{
				PublicIP: "1.2.3.4",
			},
			expected: "",
		},
		{
			name:     "empty instance",
			instance: &models.Instance{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.instance.GetConnectionString(); got != tt.expected {
				t.Errorf("Instance.GetConnectionString() = %v, want %v", got, tt.expected)
			}
		})
	}
}
