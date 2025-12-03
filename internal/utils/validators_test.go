package utils_test

import (
	"testing"
	"time"

	"instance-manager/internal/utils"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		hasError bool
	}{
		{
			name:     "standard go duration",
			input:    "1h30m",
			expected: 1*time.Hour + 30*time.Minute,
			hasError: false,
		},
		{
			name:     "minutes only",
			input:    "30m",
			expected: 30 * time.Minute,
			hasError: false,
		},
		{
			name:     "seconds only",
			input:    "45s",
			expected: 45 * time.Second,
			hasError: false,
		},
		{
			name:     "just number (assume hours)",
			input:    "2",
			expected: 2 * time.Hour,
			hasError: false,
		},
		{
			name:     "with space - 2 hours",
			input:    "2 hours",
			expected: 2 * time.Hour,
			hasError: false,
		},
		{
			name:     "with space - 30 minutes",
			input:    "30 minutes",
			expected: 30 * time.Minute,
			hasError: false,
		},
		{
			name:     "with space - 45 seconds",
			input:    "45 seconds",
			expected: 45 * time.Second,
			hasError: false,
		},
		{
			name:     "with space - 1 day",
			input:    "1 day",
			expected: 24 * time.Hour,
			hasError: false,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ParseDuration(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseDuration(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			input:    30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutes only",
			input:    45 * time.Minute,
			expected: "45m",
		},
		{
			name:     "hours only",
			input:    2 * time.Hour,
			expected: "2h",
		},
		{
			name:     "hours and minutes",
			input:    2*time.Hour + 30*time.Minute,
			expected: "2h30m",
		},
		{
			name:     "days only",
			input:    3 * 24 * time.Hour,
			expected: "3d",
		},
		{
			name:     "days and hours",
			input:    2*24*time.Hour + 5*time.Hour,
			expected: "2d5h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateInstanceType(t *testing.T) {
	tests := []struct {
		name         string
		instanceType string
		hasError     bool
	}{
		{
			name:         "valid t2.nano",
			instanceType: "t2.nano",
			hasError:     false,
		},
		{
			name:         "valid t2.micro",
			instanceType: "t2.micro",
			hasError:     false,
		},
		{
			name:         "valid m5.large",
			instanceType: "m5.large",
			hasError:     false,
		},
		{
			name:         "invalid type",
			instanceType: "invalid.type",
			hasError:     true,
		},
		{
			name:         "empty string",
			instanceType: "",
			hasError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateInstanceType(tt.instanceType)
			if tt.hasError {
				if err == nil {
					t.Errorf("ValidateInstanceType(%q) expected error, got nil", tt.instanceType)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateInstanceType(%q) unexpected error: %v", tt.instanceType, err)
				}
			}
		})
	}
}

func TestValidateAvailabilityZone(t *testing.T) {
	tests := []struct {
		name     string
		az       string
		hasError bool
	}{
		{
			name:     "valid us-east-1a",
			az:       "us-east-1a",
			hasError: false,
		},
		{
			name:     "valid eu-west-1b",
			az:       "eu-west-1b",
			hasError: false,
		},
		{
			name:     "valid ap-southeast-1c",
			az:       "ap-southeast-1c",
			hasError: false,
		},
		{
			name:     "invalid format",
			az:       "invalid",
			hasError: true,
		},
		{
			name:     "empty string",
			az:       "",
			hasError: true,
		},
		{
			name:     "too short",
			az:       "us-1",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateAvailabilityZone(tt.az)
			if tt.hasError {
				if err == nil {
					t.Errorf("ValidateAvailabilityZone(%q) expected error, got nil", tt.az)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAvailabilityZone(%q) unexpected error: %v", tt.az, err)
				}
			}
		})
	}
}
