package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"instance-manager/pkg/models"
)

// FileStorage implements instance storage using a JSON file
type FileStorage struct {
	filePath string
	mutex    sync.RWMutex
}

// NewFileStorage creates a new file storage instance
func NewFileStorage(filePath string) *FileStorage {
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			filePath = "/tmp/instance-manager.json"
		} else {
			filePath = filepath.Join(homeDir, ".instance-manager", "instances.json")
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	_ = os.MkdirAll(dir, 0755)

	return &FileStorage{
		filePath: filePath,
	}
}

// StorageRecord represents the structure stored in the file
type StorageRecord struct {
	Instances map[string]*models.InstanceRecord `json:"instances"`
	UpdatedAt time.Time                         `json:"updated_at"`
}

// SaveInstance saves an instance record to storage
func (fs *FileStorage) SaveInstance(instance *models.Instance) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	record := &models.InstanceRecord{
		Instance:  instance,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := fs.loadData()
	if err != nil {
		data = &StorageRecord{
			Instances: make(map[string]*models.InstanceRecord),
		}
	}

	data.Instances[instance.ID] = record
	data.UpdatedAt = time.Now()

	return fs.saveData(data)
}

// GetInstance retrieves an instance record from storage
func (fs *FileStorage) GetInstance(instanceID string) (*models.Instance, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	data, err := fs.loadData()
	if err != nil {
		return nil, err
	}

	record, exists := data.Instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	return record.Instance, nil
}

// ListInstances returns all stored instances
func (fs *FileStorage) ListInstances() ([]*models.Instance, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	data, err := fs.loadData()
	if err != nil {
		return []*models.Instance{}, nil // Return empty slice if file doesn't exist
	}

	var instances []*models.Instance
	for _, record := range data.Instances {
		instances = append(instances, record.Instance)
	}

	return instances, nil
}

// UpdateInstance updates an instance record in storage
func (fs *FileStorage) UpdateInstance(instance *models.Instance) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	data, err := fs.loadData()
	if err != nil {
		return err
	}

	record, exists := data.Instances[instance.ID]
	if !exists {
		return fmt.Errorf("instance %s not found", instance.ID)
	}

	record.Instance = instance
	record.UpdatedAt = time.Now()
	data.UpdatedAt = time.Now()

	return fs.saveData(data)
}

// DeleteInstance removes an instance record from storage
func (fs *FileStorage) DeleteInstance(instanceID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	data, err := fs.loadData()
	if err != nil {
		return err
	}

	delete(data.Instances, instanceID)
	data.UpdatedAt = time.Now()

	return fs.saveData(data)
}

// GetExpiredInstances returns instances that have exceeded their duration
func (fs *FileStorage) GetExpiredInstances() ([]*models.Instance, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	data, err := fs.loadData()
	if err != nil {
		return []*models.Instance{}, nil
	}

	var expiredInstances []*models.Instance
	for _, record := range data.Instances {
		if record.Instance.IsExpired() {
			expiredInstances = append(expiredInstances, record.Instance)
		}
	}

	return expiredInstances, nil
}

// loadData loads data from the storage file
func (fs *FileStorage) loadData() (*StorageRecord, error) {
	if _, err := os.Stat(fs.filePath); os.IsNotExist(err) {
		return &StorageRecord{
			Instances: make(map[string]*models.InstanceRecord),
		}, nil
	}

	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage file: %w", err)
	}

	var record StorageRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage data: %w", err)
	}

	if record.Instances == nil {
		record.Instances = make(map[string]*models.InstanceRecord)
	}

	return &record, nil
}

// saveData saves data to the storage file
func (fs *FileStorage) saveData(data *StorageRecord) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage data: %w", err)
	}

	err = os.WriteFile(fs.filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write storage file: %w", err)
	}

	return nil
}
