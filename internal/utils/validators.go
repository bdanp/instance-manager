package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration parses a duration string with support for common units
func ParseDuration(durationStr string) (time.Duration, error) {
	// Handle common duration formats
	durationStr = strings.ToLower(strings.TrimSpace(durationStr))

	// If it's just a number, assume hours
	if val, err := strconv.Atoi(durationStr); err == nil {
		return time.Duration(val) * time.Hour, nil
	}

	// Try parsing as standard Go duration
	duration, err := time.ParseDuration(durationStr)
	if err == nil {
		return duration, nil
	}

	// Handle custom formats like "2 hours", "30 minutes", etc.
	parts := strings.Fields(durationStr)
	if len(parts) == 2 {
		val, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid duration value: %s", parts[0])
		}

		unit := parts[1]
		switch {
		case strings.HasPrefix(unit, "second"):
			return time.Duration(val) * time.Second, nil
		case strings.HasPrefix(unit, "minute"):
			return time.Duration(val) * time.Minute, nil
		case strings.HasPrefix(unit, "hour"):
			return time.Duration(val) * time.Hour, nil
		case strings.HasPrefix(unit, "day"):
			return time.Duration(val) * 24 * time.Hour, nil
		default:
			return 0, fmt.Errorf("unknown duration unit: %s", unit)
		}
	}

	return 0, fmt.Errorf("invalid duration format: %s", durationStr)
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, hours)
}

// ValidateInstanceType checks if the instance type is valid
func ValidateInstanceType(instanceType string) error {
	validTypes := map[string]bool{
		"t2.nano":     true,
		"t2.micro":    true,
		"t2.small":    true,
		"t2.medium":   true,
		"t2.large":    true,
		"t2.xlarge":   true,
		"t2.2xlarge":  true,
		"t3.nano":     true,
		"t3.micro":    true,
		"t3.small":    true,
		"t3.medium":   true,
		"t3.large":    true,
		"t3.xlarge":   true,
		"t3.2xlarge":  true,
		"m5.large":    true,
		"m5.xlarge":   true,
		"m5.2xlarge":  true,
		"m5.4xlarge":  true,
		"m5.8xlarge":  true,
		"m5.12xlarge": true,
		"m5.16xlarge": true,
		"m5.24xlarge": true,
		"c5.large":    true,
		"c5.xlarge":   true,
		"c5.2xlarge":  true,
		"c5.4xlarge":  true,
		"c5.9xlarge":  true,
		"c5.12xlarge": true,
		"c5.18xlarge": true,
		"c5.24xlarge": true,
	}

	if !validTypes[instanceType] {
		return fmt.Errorf("invalid instance type: %s", instanceType)
	}

	return nil
}

// ValidateAvailabilityZone checks if the availability zone format is valid
func ValidateAvailabilityZone(az string) error {
	if az == "" {
		return fmt.Errorf("availability zone cannot be empty")
	}

	// Basic validation for AWS AZ format (e.g., us-east-1a)
	parts := strings.Split(az, "-")
	if len(parts) < 3 {
		return fmt.Errorf("invalid availability zone format: %s", az)
	}

	// Check if it ends with a letter
	lastPart := parts[len(parts)-1]
	if len(lastPart) < 2 {
		return fmt.Errorf("invalid availability zone format: %s", az)
	}

	return nil
}
