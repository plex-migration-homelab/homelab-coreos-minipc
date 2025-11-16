package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Markers manages completion marker files
type Markers struct {
	dir string
}

// NewMarkers creates a new Markers instance
func NewMarkers(dir string) *Markers {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/var/home/core" // Fallback for CoreOS
		}
		dir = filepath.Join(home, ".local", "homelab-setup")
	}

	return &Markers{
		dir: dir,
	}
}

// validateMarkerName ensures the marker name is safe and doesn't contain path traversal characters
func validateMarkerName(name string) error {
	if name == "" {
		return fmt.Errorf("marker name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("marker name cannot contain path separators: %s", name)
	}
	if name == ".." || name == "." {
		return fmt.Errorf("marker name cannot be '.' or '..': %s", name)
	}
	return nil
}

// Create creates a marker file
// This operation is idempotent - it succeeds even if the marker already exists
func (m *Markers) Create(name string) error {
	// Validate marker name to prevent path traversal
	if err := validateMarkerName(name); err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return fmt.Errorf("failed to create marker directory: %w", err)
	}

	markerPath := filepath.Join(m.dir, name)
	file, err := os.Create(markerPath)
	if err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}
	defer file.Close()

	return nil
}

// CreateIfNotExists atomically creates a marker file only if it doesn't exist
// Returns (wasCreated, error) where wasCreated indicates if this call created the marker
func (m *Markers) CreateIfNotExists(name string) (bool, error) {
	// Validate marker name to prevent path traversal
	if err := validateMarkerName(name); err != nil {
		return false, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create marker directory: %w", err)
	}

	markerPath := filepath.Join(m.dir, name)
	// O_CREATE|O_EXCL will fail if file already exists (atomic check-and-create)
	file, err := os.OpenFile(markerPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			// Marker already exists, this is not an error
			return false, nil
		}
		return false, fmt.Errorf("failed to create marker file: %w", err)
	}
	defer file.Close()

	return true, nil
}

// Exists checks if a marker file exists
// Returns (exists, error) where error indicates a problem checking (e.g., permission denied)
// If error is not nil, the exists value should not be trusted
func (m *Markers) Exists(name string) (bool, error) {
	// Validate marker name to prevent path traversal
	if err := validateMarkerName(name); err != nil {
		return false, err
	}

	markerPath := filepath.Join(m.dir, name)
	_, err := os.Stat(markerPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// Other error (permission denied, I/O error, etc.)
	return false, fmt.Errorf("failed to check marker existence: %w", err)
}

// Remove deletes a marker file
func (m *Markers) Remove(name string) error {
	// Validate marker name to prevent path traversal
	if err := validateMarkerName(name); err != nil {
		return err
	}

	markerPath := filepath.Join(m.dir, name)
	err := os.Remove(markerPath)
	if os.IsNotExist(err) {
		return nil // Not an error if it doesn't exist
	}
	return err
}

// RemoveAll removes all marker files
func (m *Markers) RemoveAll() error {
	if _, err := os.Stat(m.dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to remove
	}

	return os.RemoveAll(m.dir)
}

// List returns all marker names
func (m *Markers) List() ([]string, error) {
	if _, err := os.Stat(m.dir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read marker directory: %w", err)
	}

	var markers []string
	for _, entry := range entries {
		if !entry.IsDir() {
			markers = append(markers, entry.Name())
		}
	}

	return markers, nil
}

// Dir returns the marker directory path
func (m *Markers) Dir() string {
	return m.dir
}

// MarkFailed creates a failure marker for a step
func (m *Markers) MarkFailed(name string) error {
	return m.Create(name + "-failed")
}

// ClearFailure removes a failure marker
func (m *Markers) ClearFailure(name string) error {
	return m.Remove(name + "-failed")
}

// IsFailed checks if a step has a failure marker
func (m *Markers) IsFailed(name string) (bool, error) {
	return m.Exists(name + "-failed")
}

// StepStatus represents the status of a setup step
type StepStatus int

const (
	StepNotStarted StepStatus = iota
	StepCompleted
	StepFailed
)

// GetStatus returns the status of a step based on markers
func (m *Markers) GetStatus(name string) (StepStatus, error) {
	// Check for failure marker first
	failed, err := m.IsFailed(name)
	if err != nil {
		return StepNotStarted, err
	}
	if failed {
		return StepFailed, nil
	}

	// Check for completion marker
	completed, err := m.Exists(name)
	if err != nil {
		return StepNotStarted, err
	}
	if completed {
		return StepCompleted, nil
	}

	return StepNotStarted, nil
}
