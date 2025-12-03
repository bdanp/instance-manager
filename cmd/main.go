package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"instance-manager/internal/scheduler"
	"instance-manager/internal/utils"
	"instance-manager/pkg/aws"
	"instance-manager/pkg/cloud"
	"instance-manager/pkg/config"
	"instance-manager/pkg/models"
	"instance-manager/pkg/storage"
	"instance-manager/pkg/webserver"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	instanceType     string
	duration         string
	publicKeyPath    string
	availabilityZone string
	instanceID       string
	provider         string // Add provider flag
	verbose          bool
	logLevel         string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "instance-manager",
		Short: "AWS EC2 instance management tool",
		Long:  "A tool for creating and managing AWS EC2 instances with automatic lifecycle management",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				log.SetOutput(os.Stdout)
			}
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// Create command
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new EC2 instance",
		Long:  "Create a new EC2 instance with the specified configuration",
		RunE:  runCreate,
	}

	createCmd.Flags().StringVarP(&instanceType, "instance-type", "t", "t2.nano", "EC2 instance type")
	createCmd.Flags().StringVarP(&duration, "duration", "d", "1h", "Instance runtime duration (e.g., 1h, 30m, 2h30m)")
	createCmd.Flags().StringVarP(&publicKeyPath, "public-key", "k", "", "Path to SSH public key file (required)")
	createCmd.Flags().StringVarP(&availabilityZone, "availability-zone", "z", "us-east-1a", "AWS availability zone")
	createCmd.Flags().StringVarP(&provider, "provider", "P", "aws", "Cloud provider (aws, gcp)")
	if err := createCmd.MarkFlagRequired("public-key"); err != nil {
		log.Fatal(err)
	}

	// Status command
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check instance status",
		Long:  "Check the status of a specific instance",
		RunE:  runStatus,
	}

	statusCmd.Flags().StringVarP(&instanceID, "instance-id", "i", "", "Instance ID to check (required)")
	if err := statusCmd.MarkFlagRequired("instance-id"); err != nil {
		log.Fatal(err)
	}

	// List command
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all managed instances",
		Long:  "List all instances managed by this tool",
		RunE:  runList,
	}

	// Stop command
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop an instance",
		Long:  "Stop (terminate) a specific instance",
		RunE:  runStop,
	}

	stopCmd.Flags().StringVarP(&instanceID, "instance-id", "i", "", "Instance ID to stop (required)")
	if err := stopCmd.MarkFlagRequired("instance-id"); err != nil {
		log.Fatal(err)
	}

	// Show command
	var showCmd = &cobra.Command{
		Use:   "show",
		Short: "Show stored instance data",
		Long:  "Show detailed stored data for instances including communication details",
		RunE:  runShow,
	}

	showCmd.Flags().StringVarP(&instanceID, "instance-id", "i", "", "Instance ID to show (optional, shows all if not provided)")

	// Sync command
	var syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync stored data with AWS",
		Long:  "Sync stored instance data with current AWS state (updates IPs, states, etc.)",
		RunE:  runSync,
	}

	syncCmd.Flags().StringVarP(&instanceID, "instance-id", "i", "", "Instance ID to sync (optional, syncs all if not provided)")

	// Extend command
	var extendCmd = &cobra.Command{
		Use:   "extend",
		Short: "Extend instance TTL",
		Long:  "Extend the TTL (time-to-live) of an existing instance",
		RunE:  runExtend,
	}

	extendCmd.Flags().StringVarP(&instanceID, "instance-id", "i", "", "Instance ID to extend (required)")
	extendCmd.Flags().StringVarP(&duration, "duration", "d", "", "Additional duration to extend (e.g., 1h, 30m, 2h30m) (required)")
	if err := extendCmd.MarkFlagRequired("instance-id"); err != nil {
		log.Fatal(err)
	}
	if err := extendCmd.MarkFlagRequired("duration"); err != nil {
		log.Fatal(err)
	}

	// Service command (enhanced scheduler)
	var serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Run background service",
		Long:  "Run the background service to monitor instance lifecycle, handle TTL changes, and manage instance state",
		RunE:  runService,
	}

	// Web command
	var webPort int
	var webCmd = &cobra.Command{
		Use:   "web",
		Short: "Start web server",
		Long:  "Start a web server for managing AWS instances through a browser interface",
		RunE:  runWeb,
	}

	webCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Port to run the web server on")

	// Terminate command
	var terminateCmd = &cobra.Command{
		Use:   "terminate",
		Short: "Terminate an instance (permanently deletes it)",
		Long:  "Terminate a specific instance. This action cannot be undone.",
		RunE:  runTerminate,
	}
	var terminateInstanceID string
	terminateCmd.Flags().StringVarP(&terminateInstanceID, "instance-id", "i", "", "Instance ID to terminate (required)")
	if err := terminateCmd.MarkFlagRequired("instance-id"); err != nil {
		log.Fatal(err)
	}

	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(extendCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(terminateCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate inputs
	if err := config.ValidatePublicKeyPath(publicKeyPath); err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	if err := utils.ValidateInstanceType(instanceType); err != nil {
		return fmt.Errorf("invalid instance type: %w", err)
	}

	if err := utils.ValidateAvailabilityZone(availabilityZone); err != nil {
		return fmt.Errorf("invalid availability zone: %w", err)
	}

	parsedDuration, err := utils.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	// Create provider based on flag
	var cloudProvider cloud.CloudProvider
	switch provider {
	case "aws":
		cloudProvider, err = aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to create AWS provider: %w", err)
		}
	// case "gcp":
	// 	cloudProvider, err = gcp.NewProvider(...)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to create GCP provider: %w", err)
	// 	}
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	// Validate credentials
	if err := cloudProvider.ValidateCredentials(); err != nil {
		return fmt.Errorf("failed to validate AWS credentials: %w", err)
	}

	// Create instance configuration
	instanceConfig := models.InstanceConfig{
		InstanceType:     instanceType,
		Duration:         parsedDuration,
		PublicKeyPath:    publicKeyPath,
		AvailabilityZone: availabilityZone,
		Region:           cfg.AWS.Region,
	}

	fmt.Printf("Creating instance with configuration:\n")
	fmt.Printf("  Instance Type: %s\n", instanceConfig.InstanceType)
	fmt.Printf("  Duration: %s\n", utils.FormatDuration(instanceConfig.Duration))
	fmt.Printf("  Public Key: %s\n", instanceConfig.PublicKeyPath)
	fmt.Printf("  Availability Zone: %s\n", instanceConfig.AvailabilityZone)
	fmt.Printf("\nCreating instance...\n")

	// Create instance
	instance, err := cloudProvider.CreateInstance(instanceConfig)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	// Save instance to storage
	storage := storage.NewFileStorage("")
	if err := storage.SaveInstance(instance); err != nil {
		log.Printf("Warning: failed to save instance to storage: %v", err)
	}

	fmt.Printf("\nInstance created successfully!\n")
	fmt.Printf("  Instance ID: %s\n", instance.ID)
	fmt.Printf("  State: %s\n", instance.State)
	fmt.Printf("  Expires at: %s\n", instance.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("\nUse 'instance-manager status --instance-id %s' to check status\n", instance.ID)

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create AWS provider
	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	// Get instance status
	status, err := provider.GetInstanceStatus(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance status: %w", err)
	}

	fmt.Printf("Instance Status:\n")
	fmt.Printf("  ID: %s\n", status.ID)
	fmt.Printf("  State: %s\n", status.State)
	fmt.Printf("  Ready: %t\n", status.Ready)

	if status.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", status.PublicIP)
		fmt.Printf("  SSH Command: ssh %s@%s\n", status.Username, status.PublicIP)
	}

	if status.PrivateIP != "" {
		fmt.Printf("  Private IP: %s\n", status.PrivateIP)
	}

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create AWS provider
	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	// List instances
	instances, err := provider.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(instances) == 0 {
		fmt.Println("No managed instances found.")
		return nil
	}

	fmt.Printf("Managed Instances:\n\n")
	for _, instance := range instances {
		fmt.Printf("Instance ID: %s\n", instance.ID)
		fmt.Printf("  Type: %s\n", instance.InstanceType)
		fmt.Printf("  State: %s\n", instance.State)
		fmt.Printf("  Launch Time: %s\n", instance.LaunchTime.Format(time.RFC3339))
		fmt.Printf("  Duration: %s\n", utils.FormatDuration(instance.Duration))
		fmt.Printf("  Expires At: %s\n", instance.ExpiresAt.Format(time.RFC3339))
		fmt.Printf("  Availability Zone: %s\n", instance.AvailabilityZone)

		if instance.PublicIP != "" {
			fmt.Printf("  Public IP: %s\n", instance.PublicIP)
			fmt.Printf("  SSH Command: ssh %s@%s\n", instance.Username, instance.PublicIP)
		}

		if instance.IsExpired() {
			fmt.Printf("  Status: EXPIRED\n")
		} else {
			timeLeft := time.Until(instance.ExpiresAt)
			fmt.Printf("  Time Remaining: %s\n", utils.FormatDuration(timeLeft))
		}

		fmt.Println()
	}

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create AWS provider
	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	fmt.Printf("Stopping instance %s...\n", instanceID)

	// Terminate instance
	if err := provider.TerminateInstance(instanceID); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	// Update storage
	storage := storage.NewFileStorage("")
	instance, err := storage.GetInstance(instanceID)
	if err == nil {
		instance.State = "terminated"
		if err := storage.UpdateInstance(instance); err != nil {
			log.Printf("Warning: failed to update instance state in storage: %v", err)
		}
	}

	fmt.Printf("Instance %s has been stopped.\n", instanceID)
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	// Create storage
	storage := storage.NewFileStorage("")

	if instanceID == "" {
		// Show all instances
		instances, err := storage.ListInstances()
		if err != nil {
			return fmt.Errorf("failed to load instances: %w", err)
		}

		if len(instances) == 0 {
			fmt.Println("No instances found in storage.")
			fmt.Println("Create an instance first using: instance-manager create --public-key ~/.ssh/id_rsa.pub")
			return nil
		}

		fmt.Printf("=== All Stored Instances (%d total) ===\n\n", len(instances))
		for i, instance := range instances {
			fmt.Printf("Instance %d:\n", i+1)
			printDetailedInstanceInfo(instance)
			fmt.Println()
		}
	} else {
		// Show specific instance
		instance, err := storage.GetInstance(instanceID)
		if err != nil {
			return fmt.Errorf("instance %s not found: %w", instanceID, err)
		}

		fmt.Printf("=== Instance Communication Details ===\n\n")
		printDetailedInstanceInfo(instance)
	}
	return nil
}

func printDetailedInstanceInfo(instance *models.Instance) {
	fmt.Printf("🆔 Instance ID: %s\n", instance.ID)
	fmt.Printf("💻 Instance Type: %s\n", instance.InstanceType)
	fmt.Printf("📍 Availability Zone: %s\n", instance.AvailabilityZone)
	fmt.Printf("🔑 Key Name: %s\n", instance.KeyName)
	fmt.Printf("👤 Username: %s\n", instance.Username)

	fmt.Printf("\n🌐 Network & Communication Details:\n")
	if instance.PublicIP != "" {
		fmt.Printf("   📡 Public IP: %s\n", instance.PublicIP)
		fmt.Printf("   🔗 SSH Command: ssh -i ~/.ssh/id_rsa %s@%s\n", instance.Username, instance.PublicIP)
		fmt.Printf("   🌍 Web Access: http://%s (if web server running)\n", instance.PublicIP)
	} else {
		fmt.Printf("   📡 Public IP: Not assigned yet (instance may be starting)\n")
		fmt.Printf("   💡 Tip: Check status again in a few minutes\n")
	}

	if instance.PrivateIP != "" {
		fmt.Printf("   🏠 Private IP: %s\n", instance.PrivateIP)
	}

	fmt.Printf("\n📊 Instance Status:\n")
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Launch Time: %s\n", instance.LaunchTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Duration: %s\n", utils.FormatDuration(instance.Duration))
	fmt.Printf("   Expires At: %s\n", instance.ExpiresAt.Format("2006-01-02 15:04:05"))

	if instance.IsExpired() {
		fmt.Printf("   ⚠️  Status: EXPIRED\n")
		fmt.Printf("   💡 Tip: This instance should be terminated automatically\n")
	} else {
		timeLeft := time.Until(instance.ExpiresAt)
		fmt.Printf("   ⏳ Time Remaining: %s\n", utils.FormatDuration(timeLeft))

		if timeLeft < 10*time.Minute {
			fmt.Printf("   ⚠️  Warning: Instance will expire soon!\n")
			fmt.Printf("   💡 Extend with: instance-manager extend --instance-id %s --duration 1h\n", instance.ID)
		}
	}

	// Show full connection string if available
	connectionString := instance.GetConnectionString()
	if connectionString != "" {
		fmt.Printf("\n🚀 Quick Connect: %s\n", connectionString)
	}

	fmt.Printf("\n📝 Storage Information:\n")
	fmt.Printf("   📁 This data is persisted locally for future access\n")
	fmt.Printf("   🔄 Run 'instance-manager service' for automatic lifecycle management\n")
}

func runExtend(cmd *cobra.Command, args []string) error {
	// Parse duration
	parsedDuration, err := utils.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	// Create storage
	storage := storage.NewFileStorage("")

	// Get instance
	instance, err := storage.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Extend TTL
	oldExpiresAt := instance.ExpiresAt
	instance.ExpiresAt = instance.ExpiresAt.Add(parsedDuration)
	instance.Duration = instance.Duration + parsedDuration

	// Update storage
	if err := storage.UpdateInstance(instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	fmt.Printf("Instance TTL extended successfully!\n")
	fmt.Printf("  Instance ID: %s\n", instance.ID)
	fmt.Printf("  Previous expiry: %s\n", oldExpiresAt.Format(time.RFC3339))
	fmt.Printf("  New expiry: %s\n", instance.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("  Extended by: %s\n", utils.FormatDuration(parsedDuration))

	// If the instance is currently stopped and the new TTL is in the future,
	// let the user know that the service will restart it
	if instance.State == "stopped" && instance.ExpiresAt.After(time.Now()) {
		fmt.Printf("\nNote: Instance is currently stopped. The background service will automatically start it.\n")
		fmt.Printf("To manually start the service: %s service --log-level info\n", os.Args[0])
	}

	return nil
}

func runSync(cmd *cobra.Command, args []string) error {
	// Get the instance ID from the flag
	syncInstanceID, _ := cmd.Flags().GetString("instance-id")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create AWS provider
	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	// Create storage
	storage := storage.NewFileStorage("")

	// Sync all instances if no specific ID is provided
	if syncInstanceID == "" {
		instances, err := storage.ListInstances()
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}

		for _, instance := range instances {
			awsProvider, ok := provider.(*aws.Provider)
			if !ok {
				return fmt.Errorf("invalid provider type for sync operation")
			}
			if err := syncInstanceData(awsProvider, storage, instance.ID); err != nil {
				log.Printf("Warning: failed to sync instance %s: %v", instance.ID, err)
			}
		}

		fmt.Println("Sync completed for all instances.")
	} else {
		// Sync specific instance
		awsProvider, ok := provider.(*aws.Provider)
		if !ok {
			return fmt.Errorf("invalid provider type for sync operation")
		}
		if err := syncInstanceData(awsProvider, storage, syncInstanceID); err != nil {
			return fmt.Errorf("failed to sync instance %s: %w", syncInstanceID, err)
		}

		fmt.Printf("Sync completed for instance %s.\n", syncInstanceID)
	}

	return nil
}

func syncInstanceData(provider *aws.Provider, storage *storage.FileStorage, instanceID string) error {
	// Get current instance data from AWS
	currentData, err := provider.GetInstanceStatus(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance status from AWS: %w", err)
	}

	// Get stored instance data
	storedInstance, err := storage.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance from storage: %w", err)
	}

	// Update stored instance data with current data
	storedInstance.PublicIP = currentData.PublicIP
	storedInstance.PrivateIP = currentData.PrivateIP
	storedInstance.State = currentData.State
	// Note: Ready status is determined by PublicIP presence and state

	// Update storage
	if err := storage.UpdateInstance(storedInstance); err != nil {
		return fmt.Errorf("failed to update instance in storage: %w", err)
	}

	fmt.Printf("Instance %s synced: PublicIP=%s, State=%s\n", instanceID, storedInstance.PublicIP, storedInstance.State)
	return nil
}

// getLogLevel parses log level string to logrus level
func getLogLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

func runService(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create AWS provider
	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	// Validate credentials
	if err := provider.ValidateCredentials(); err != nil {
		return fmt.Errorf("failed to validate AWS credentials: %w", err)
	}

	// Create storage
	storage := storage.NewFileStorage("")

	// Create and configure scheduler
	scheduler := scheduler.NewScheduler(provider, storage)

	// Set log level
	logLevelParsed := getLogLevel(logLevel)
	if verbose {
		logLevelParsed = logrus.DebugLevel
	}
	scheduler.SetLogLevel(logLevelParsed)

	// Start scheduler
	scheduler.Start()

	fmt.Printf("Instance Manager service started (log level: %s)\n", logLevel)
	fmt.Println("Monitoring instance lifecycle, TTL changes, and state management...")
	fmt.Println("Press Ctrl+C to stop the service.")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	scheduler.Stop()
	fmt.Println("Service stopped.")
	return nil
}
func runWeb(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create AWS provider
	provider, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	// Validate credentials
	if err := provider.ValidateCredentials(); err != nil {
		return fmt.Errorf("failed to validate AWS credentials: %w", err)
	}

	// Create storage
	storage := storage.NewFileStorage("")

	// Create logger
	logger := logrus.New()
	logger.SetLevel(getLogLevel(logLevel))
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}
	logger.SetOutput(os.Stdout)

	// Create and start web server
	webPort, _ := cmd.Flags().GetInt("port")
	server := webserver.NewServer(provider, storage, logger, webPort)

	fmt.Printf("AWS Instance Manager Web Server starting on http://localhost:%d\n", webPort)
	fmt.Println("Open your browser and navigate to the address above.")
	fmt.Println("Press Ctrl+C to stop the server.")

	return server.Start()
}

func runTerminate(cmd *cobra.Command, args []string) error {
	instanceID, err := cmd.Flags().GetString("instance-id")
	if err != nil {
		return err
	}
	provider, storage, err := getProviderAndStorage()
	if err != nil {
		return err
	}
	fmt.Printf("Terminating instance %s...\n", instanceID)
	err = provider.TerminateInstance(instanceID)
	if err != nil {
		return fmt.Errorf("Failed to terminate instance: %w", err)
	}
	// Remove from storage
	_ = storage.DeleteInstance(instanceID)
	fmt.Printf("Instance %s has been terminated and removed from storage.\n", instanceID)
	return nil
}

func getProviderAndStorage() (*aws.Provider, *storage.FileStorage, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	providerIface, err := aws.NewProvider(cfg.AWS.Region, cfg.AWS.AccessKey, cfg.AWS.SecretKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AWS provider: %w", err)
	}
	provider, ok := providerIface.(*aws.Provider)
	if !ok {
		return nil, nil, fmt.Errorf("provider type assertion failed")
	}
	if err := provider.ValidateCredentials(); err != nil {
		return nil, nil, fmt.Errorf("failed to validate AWS credentials: %w", err)
	}
	storage := storage.NewFileStorage("")
	return provider, storage, nil
}
