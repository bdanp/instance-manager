package scheduler

import (
	"context"
	"time"

	"instance-manager/pkg/cloud"
	"instance-manager/pkg/models"
	"instance-manager/pkg/storage"

	"github.com/sirupsen/logrus"
)

// Scheduler manages background tasks for instance lifecycle
type Scheduler struct {
	provider       cloud.CloudProvider
	storage        *storage.FileStorage
	interval       time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *logrus.Logger
	lastReload     time.Time
	reloadInterval time.Duration
}

// NewScheduler creates a new scheduler instance
func NewScheduler(provider cloud.CloudProvider, storage *storage.FileStorage) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logger.SetLevel(logrus.InfoLevel)

	return &Scheduler{
		provider:       provider,
		storage:        storage,
		interval:       30 * time.Second, // Check every 30 seconds for better responsiveness
		reloadInterval: 10 * time.Second, // Reload data every 10 seconds max
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
		lastReload:     time.Time{}, // Force initial reload
	}
}

// SetLogLevel sets the logging level
func (s *Scheduler) SetLogLevel(level logrus.Level) {
	s.logger.SetLevel(level)
}

// Start begins the background scheduler
func (s *Scheduler) Start() {
	s.logger.WithFields(logrus.Fields{
		"interval":        s.interval,
		"reload_interval": s.reloadInterval,
	}).Info("Starting instance scheduler")
	go s.run()
}

// Stop stops the background scheduler
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping instance scheduler")
	s.cancel()
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Scheduler stopped")
			return
		case <-ticker.C:
			s.processInstances()
		}
	}
}

// processInstances checks all instances and takes appropriate actions
func (s *Scheduler) processInstances() {
	s.logger.Debug("Processing instances...")

	// Get all instances from storage (this will reload if needed)
	instances, err := s.getInstancesWithReload()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get instances from storage")
		return
	}

	s.logger.WithField("instance_count", len(instances)).Debug("Loaded instances from storage")

	for _, instance := range instances {
		s.processInstance(instance)
	}
}

// getInstancesWithReload gets instances and ensures data is fresh (max 10 seconds old)
func (s *Scheduler) getInstancesWithReload() ([]*models.Instance, error) {
	// Force reload if data is older than reloadInterval
	if time.Since(s.lastReload) > s.reloadInterval {
		s.logger.Debug("Reloading instance data from storage")
		s.lastReload = time.Now()
	}

	return s.storage.ListInstances()
}

// processInstance handles the lifecycle of a single instance
func (s *Scheduler) processInstance(instance *models.Instance) {
	logger := s.logger.WithFields(logrus.Fields{
		"instance_id": instance.ID,
		"state":       instance.State,
		"expires_at":  instance.ExpiresAt,
	})

	logger.Debug("Processing instance")

	// Skip if instance is already terminated
	if instance.State == "terminated" || instance.State == "terminating" {
		logger.Debug("Instance already terminated, skipping")
		return
	}

	// Get current instance status from cloud provider
	status, err := s.provider.GetInstanceStatus(instance.ID)
	if err != nil {
		logger.WithError(err).Warn("Failed to get instance status from cloud provider")
		return
	}

	// Update local state if it differs from cloud state
	if status.State != instance.State {
		logger.WithFields(logrus.Fields{
			"old_state": instance.State,
			"new_state": status.State,
		}).Info("Instance state changed, updating local storage")

		instance.State = status.State
		instance.PublicIP = status.PublicIP
		instance.PrivateIP = status.PrivateIP

		if err := s.storage.UpdateInstance(instance); err != nil {
			logger.WithError(err).Error("Failed to update instance in storage")
		}
	}

	// Check if instance has expired and should be stopped
	if instance.IsExpired() {
		// Only stop if instance is currently running or pending
		if status.State == "running" || status.State == "pending" {
			s.handleExpiredInstance(instance, logger)
		} else {
			logger.Debug("Instance expired but already stopped/terminated")
		}
		return
	}

	// Check if instance should be started (if TTL was extended and instance is stopped)
	if instance.ExpiresAt.After(time.Now()) && (status.State == "stopped" || status.State == "stopping") {
		s.handleStoppedInstance(instance, logger)
	}
}

// handleExpiredInstance stops an expired instance (instead of terminating)
func (s *Scheduler) handleExpiredInstance(instance *models.Instance, logger *logrus.Entry) {
	timeOverdue := time.Since(instance.ExpiresAt)

	logger.WithField("overdue_duration", timeOverdue).Warn("Instance has EXPIRED - stopping instance (can be restarted if TTL extended)")

	// Stop the instance (not terminate)
	if err := s.provider.StopInstance(instance.ID); err != nil {
		logger.WithError(err).Error("Failed to stop expired instance")
		return
	}

	// Update instance state in storage
	instance.State = "stopping"
	if err := s.storage.UpdateInstance(instance); err != nil {
		logger.WithError(err).Error("Failed to update instance state in storage")
	}

	logger.WithFields(logrus.Fields{
		"overdue_duration": timeOverdue,
		"action":           "stopped",
	}).Info("✅ Successfully stopped expired instance (can be restarted)")
}

// handleStoppedInstance starts a stopped instance if its TTL was extended
func (s *Scheduler) handleStoppedInstance(instance *models.Instance, logger *logrus.Entry) {
	timeRemaining := time.Until(instance.ExpiresAt)

	logger.WithField("time_remaining", timeRemaining).Info("Instance TTL was EXTENDED - restarting stopped instance")

	// Start the instance
	if err := s.startInstance(instance.ID); err != nil {
		logger.WithError(err).Error("Failed to start stopped instance")
		return
	}

	// Update instance state in storage
	instance.State = "pending"
	if err := s.storage.UpdateInstance(instance); err != nil {
		logger.WithError(err).Error("Failed to update instance state in storage")
	}

	logger.WithFields(logrus.Fields{
		"time_remaining": timeRemaining,
		"action":         "restarted",
	}).Info("🚀 Successfully restarted instance due to TTL extension")
}

// startInstance starts a stopped EC2 instance
func (s *Scheduler) startInstance(instanceID string) error {
	return s.provider.StartInstance(instanceID)
}

// RunOnce executes the scheduler logic once (useful for testing and manual runs)
func (s *Scheduler) RunOnce() {
	s.logger.Info("Running scheduler once")
	s.processInstances()
}
