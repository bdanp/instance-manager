package main

import (
	"fmt"
	"os"
	"time"

	"instance-manager/pkg/models"
	"instance-manager/pkg/storage"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: show-data <instance-id>")
		fmt.Println("   or: show-data all")
		os.Exit(1)
	}

	store := storage.NewFileStorage("")

	if os.Args[1] == "all" {
		instances, err := store.ListInstances()
		if err != nil {
			fmt.Printf("Error loading instances: %v\n", err)
			os.Exit(1)
		}

		if len(instances) == 0 {
			fmt.Println("No instances found in storage.")
			return
		}

		fmt.Printf("=== All Stored Instances (%d total) ===\n\n", len(instances))
		for i, instance := range instances {
			fmt.Printf("Instance %d:\n", i+1)
			printInstanceDetails(instance)
			fmt.Println()
		}

	} else {
		instanceID := os.Args[1]

		instance, err := store.GetInstance(instanceID)
		if err != nil {
			fmt.Printf("Instance %s not found: %v\n", instanceID, err)
			os.Exit(1)
		}

		fmt.Println("=== Instance Communication Details ===")
		printInstanceDetails(instance)
	}
}

func printInstanceDetails(instance *models.Instance) {
	fmt.Printf("🆔 Instance ID: %s\n", instance.ID)
	fmt.Printf("💻 Instance Type: %s\n", instance.InstanceType)
	fmt.Printf("📍 Availability Zone: %s\n", instance.AvailabilityZone)
	fmt.Printf("🔑 Key Name: %s\n", instance.KeyName)
	fmt.Printf("👤 Username: %s\n", instance.Username)

	fmt.Println("🌐 Network Details:")
	if instance.PublicIP != "" {
		fmt.Printf("   Public IP: %s\n", instance.PublicIP)
		fmt.Printf("   🔗 SSH Command: ssh %s@%s\n", instance.Username, instance.PublicIP)
	} else {
		fmt.Println("   Public IP: Not assigned yet (instance may be starting)")
	}

	if instance.PrivateIP != "" {
		fmt.Printf("   Private IP: %s\n", instance.PrivateIP)
	}

	fmt.Printf("📊 Status: %s\n", instance.State)
	fmt.Printf("⏰ Launch Time: %s\n", instance.LaunchTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("⌛ Duration: %s\n", instance.Duration)
	fmt.Printf("🔔 Expires At: %s\n", instance.ExpiresAt.Format("2006-01-02 15:04:05"))

	if instance.IsExpired() {
		fmt.Println("⚠️  Status: EXPIRED")
	} else {
		timeLeft := time.Until(instance.ExpiresAt).Round(time.Second)
		fmt.Printf("⏳ Time Remaining: %s\n", timeLeft)
	}

	// Optional: Show full connection string
	if conn := instance.GetConnectionString(); conn != "" {
		fmt.Printf("🔗 Connection String: %s\n", conn)
	}
}
