package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"instance-manager/internal/utils"
	"instance-manager/pkg/cloud"
	"instance-manager/pkg/models"
	"instance-manager/pkg/storage"

	"github.com/sirupsen/logrus"
)

// Server holds the web server state
type Server struct {
	provider cloud.CloudProvider
	storage  *storage.FileStorage
	logger   *logrus.Logger
	port     int
}

// APIResponse represents the API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// CreateInstanceRequest represents the request to create an instance
type CreateInstanceRequest struct {
	InstanceType     string `json:"instance_type"`
	Duration         string `json:"duration"`
	PublicKeyPath    string `json:"public_key_path"`
	AvailabilityZone string `json:"availability_zone"`
	Provider         string `json:"provider"` // Add provider field
}

// ExtendInstanceRequest represents the request to extend an instance
type ExtendInstanceRequest struct {
	Duration string `json:"duration"`
}

// NewServer creates a new web server instance
func NewServer(provider cloud.CloudProvider, storage *storage.FileStorage, logger *logrus.Logger, port int) *Server {
	return &Server{
		provider: provider,
		storage:  storage,
		logger:   logger,
		port:     port,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	// Setup routes
	http.HandleFunc("/api/health", s.handleHealth)
	http.HandleFunc("/api/instances", s.handleInstances)
	http.HandleFunc("/api/instances/create", s.handleCreateInstance)
	http.HandleFunc("/api/instances/status", s.handleInstanceStatus)
	http.HandleFunc("/api/instances/extend", s.handleExtendInstance)
	http.HandleFunc("/api/instances/stop", s.handleStopInstance)
	http.HandleFunc("/api/instances/terminate", s.handleTerminateInstance)

	// Serve static files
	http.HandleFunc("/", s.handleStaticFiles)

	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Infof("Starting web server on http://localhost%s", addr)
	return http.ListenAndServe(addr, nil)
}

// Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Service is healthy",
	})
}

func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	instances, err := s.storage.ListInstances()
	if err != nil {
		s.logger.WithError(err).Error("Failed to list instances")
		s.jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get instances: %v", err),
		})
		return
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].ExpiresAt.After(instances[j].ExpiresAt)
	})
	// Sync each instance with latest AWS data
	for _, instance := range instances {
		status, err := s.provider.GetInstanceStatus(instance.ID)
		if err != nil {
			s.logger.WithError(err).Debug("Failed to sync instance", map[string]interface{}{"instance_id": instance.ID})
			continue
		}

		// Update instance with latest data if changed
		if status.PublicIP != instance.PublicIP || status.PrivateIP != instance.PrivateIP || status.State != instance.State {
			instance.PublicIP = status.PublicIP
			instance.PrivateIP = status.PrivateIP
			instance.State = status.State
			instance.Username = status.Username // Also update username if available

			// Save updated instance silently
			if err := s.storage.SaveInstance(instance); err != nil {
				s.logger.WithError(err).Debug("Failed to save synced instance data")
			}
		}
	}

	s.logger.WithField("count", len(instances)).Debug("Listed instances")
	s.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d instances", len(instances)),
		Data:    instances,
	})
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	var req CreateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.WithError(err).Error("Failed to decode create request")
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Set defaults
	if req.InstanceType == "" {
		req.InstanceType = "t2.nano"
	}
	if req.Duration == "" {
		req.Duration = "1h"
	}
	if req.AvailabilityZone == "" {
		req.AvailabilityZone = "us-east-1a"
	}
	// In handleCreateInstance, set provider default if empty
	if req.Provider == "" {
		req.Provider = "aws"
	}

	// Validate public key path
	if req.PublicKeyPath == "" {
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "public_key_path is required",
		})
		return
	}

	// Validate duration
	duration, err := utils.ParseDuration(req.Duration)
	if err != nil {
		s.logger.WithError(err).Warn("Invalid duration")
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid duration: %v", err),
		})
		return
	}

	// Create instance
	config := models.InstanceConfig{
		InstanceType:     req.InstanceType,
		Duration:         duration,
		PublicKeyPath:    req.PublicKeyPath,
		AvailabilityZone: req.AvailabilityZone,
		Region:           "us-east-1", // or from config
	}

	s.logger.WithFields(map[string]interface{}{
		"type":     req.InstanceType,
		"duration": duration.String(),
		"zone":     req.AvailabilityZone,
	}).Info("Creating instance")

	instance, err := s.provider.CreateInstance(config)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create instance")
		s.jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create instance: %v", err),
		})
		return
	}

	// Store instance
	instance.Provider = req.Provider // Set provider on instance
	if err := s.storage.SaveInstance(instance); err != nil {
		s.logger.WithError(err).Error("Failed to save instance")
		s.jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save instance: %v", err),
		})
		return
	}

	s.logger.WithField("instance_id", instance.ID).Info("Instance created successfully")
	s.jsonResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Instance created successfully",
		Data:    instance,
	})
}

func (s *Server) handleInstanceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "instance_id query parameter is required",
		})
		return
	}

	instance, err := s.storage.GetInstance(instanceID)
	if err != nil {
		s.logger.WithError(err).Warn("Instance not found in storage", map[string]interface{}{"instance_id": instanceID})
		s.jsonResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Instance not found: %v", err),
		})
		return
	}

	status, err := s.provider.GetInstanceStatus(instanceID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get status from AWS", map[string]interface{}{"instance_id": instanceID})
		s.jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get instance status: %v", err),
		})
		return
	}

	// Update instance with latest data from AWS
	if status.PublicIP != instance.PublicIP || status.PrivateIP != instance.PrivateIP || status.State != instance.State {
		instance.PublicIP = status.PublicIP
		instance.PrivateIP = status.PrivateIP
		instance.State = status.State

		// Save updated instance
		if err := s.storage.SaveInstance(instance); err != nil {
			s.logger.WithError(err).Warn("Failed to sync instance data")
		} else {
			s.logger.WithField("instance_id", instanceID).Debug("Instance data synced from AWS")
		}
	}

	data := map[string]interface{}{
		"instance":       instance,
		"status":         status,
		"is_expired":     instance.IsExpired(),
		"time_remaining": time.Until(instance.ExpiresAt).Seconds(),
	}

	s.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Instance status retrieved",
		Data:    data,
	})
}

func (s *Server) handleExtendInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "instance_id query parameter is required",
		})
		return
	}

	var req ExtendInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	duration, err := utils.ParseDuration(req.Duration)
	if err != nil {
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid duration: %v", err),
		})
		return
	}

	instance, err := s.storage.GetInstance(instanceID)
	if err != nil {
		s.jsonResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Instance not found: %v", err),
		})
		return
	}

	// Extend the expiry time
	instance.ExpiresAt = instance.ExpiresAt.Add(duration)

	if err := s.storage.SaveInstance(instance); err != nil {
		s.jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to extend instance: %v", err),
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Instance TTL extended successfully",
		Data:    instance,
	})
}

func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "instance_id query parameter is required",
		})
		return
	}

	if err := s.provider.StopInstance(instanceID); err != nil {
		s.jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to stop instance: %v", err),
		})
		return
	}
	// Update ExpiresAt to now so service will not restart
	instance, err := s.storage.GetInstance(instanceID)
	if err == nil {
		instance.ExpiresAt = time.Now()
		_ = s.storage.SaveInstance(instance)
	}

	s.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Instance stopped successfully",
	})
}

func (s *Server) handleTerminateInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		s.jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "instance_id query parameter is required",
		})
		return
	}
	if err := s.provider.TerminateInstance(instanceID); err != nil {
		s.jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to terminate instance: %v", err),
		})
		return
	}
	_ = s.storage.DeleteInstance(instanceID)
	s.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Instance terminated successfully",
	})
}

func (s *Server) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, getIndexHTML())
		return
	}

	if r.URL.Path == "/css/style.css" {
		w.Header().Set("Content-Type", "text/css")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, getStyleCSS())
		return
	}

	if r.URL.Path == "/js/app.js" {
		w.Header().Set("Content-Type", "application/javascript")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, getAppJS())
		return
	}

	s.jsonResponse(w, http.StatusNotFound, APIResponse{
		Success: false,
		Error:   "Not found",
	})
}

// Helper methods

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// Content functions remain the same
