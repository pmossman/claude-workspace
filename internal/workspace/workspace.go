package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Manager handles workspace directory operations
type Manager struct {
	baseDir string
}

// NewManager creates a new workspace manager
func NewManager(baseDir string) *Manager {
	return &Manager{baseDir: baseDir}
}

// GetPath returns the full path to a workspace directory
func (m *Manager) GetPath(name string) string {
	return filepath.Join(m.baseDir, name)
}

// Create creates a new workspace directory structure
func (m *Manager) Create(name string) error {
	wsPath := m.GetPath(name)

	// Create main workspace directory
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Create research subdirectory
	researchPath := filepath.Join(wsPath, "research")
	if err := os.MkdirAll(researchPath, 0755); err != nil {
		return fmt.Errorf("failed to create research directory: %w", err)
	}

	// Create initial empty files
	files := []string{"context.md", "decisions.md", "continuation.md", "summary.txt"}
	for _, file := range files {
		filePath := filepath.Join(wsPath, file)
		if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", file, err)
		}
	}

	return nil
}

// Exists checks if a workspace directory exists
func (m *Manager) Exists(name string) bool {
	wsPath := m.GetPath(name)
	_, err := os.Stat(wsPath)
	return err == nil
}

// GetSummary reads the summary.txt file for a workspace
func (m *Manager) GetSummary(name string) string {
	summaryPath := filepath.Join(m.GetPath(name), "summary.txt")
	data, err := os.ReadFile(summaryPath)
	if err != nil || len(data) == 0 {
		return "(no summary)"
	}
	return strings.TrimSpace(string(data))
}

// GetContinuation reads the continuation.md file for a workspace
func (m *Manager) GetContinuation(name string) string {
	contPath := filepath.Join(m.GetPath(name), "continuation.md")
	data, err := os.ReadFile(contPath)
	if err != nil || len(data) == 0 {
		return ""
	}
	return string(data)
}

// GetContext reads the context.md file for a workspace
func (m *Manager) GetContext(name string) string {
	contextPath := filepath.Join(m.GetPath(name), "context.md")
	data, err := os.ReadFile(contextPath)
	if err != nil || len(data) == 0 {
		return "(no context yet)"
	}
	// Return first 200 chars for preview
	text := strings.TrimSpace(string(data))
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}

// CreateLock creates a lock file for a workspace
func (m *Manager) CreateLock(name string, pid int) error {
	lockPath := filepath.Join(m.GetPath(name), ".lock")
	content := fmt.Sprintf("%d", pid)
	return os.WriteFile(lockPath, []byte(content), 0644)
}

// RemoveLock removes the lock file for a workspace
func (m *Manager) RemoveLock(name string) error {
	lockPath := filepath.Join(m.GetPath(name), ".lock")
	err := os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CheckLock checks if a workspace is locked and if the process is still running
func (m *Manager) CheckLock(name string) (bool, int, error) {
	lockPath := filepath.Join(m.GetPath(name), ".lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false, 0, fmt.Errorf("invalid lock file: %w", err)
	}

	// Check if process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist
		return false, pid, nil
	}

	// Try to send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist or we can't signal it
		return false, pid, nil
	}

	return true, pid, nil
}

// Archive moves a workspace to an archived subdirectory
func (m *Manager) Archive(name string) error {
	wsPath := m.GetPath(name)
	archivePath := filepath.Join(m.baseDir, "archived", name)

	// Create archived directory
	if err := os.MkdirAll(filepath.Join(m.baseDir, "archived"), 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Move workspace
	if err := os.Rename(wsPath, archivePath); err != nil {
		return fmt.Errorf("failed to archive workspace: %w", err)
	}

	return nil
}

// Clone copies a workspace directory to a new name
func (m *Manager) Clone(fromName, toName string) error {
	fromPath := m.GetPath(fromName)
	toPath := m.GetPath(toName)

	// Check source exists
	if !m.Exists(fromName) {
		return fmt.Errorf("source workspace '%s' does not exist", fromName)
	}

	// Check destination doesn't exist
	if m.Exists(toName) {
		return fmt.Errorf("destination workspace '%s' already exists", toName)
	}

	// Create destination
	if err := m.Create(toName); err != nil {
		return err
	}

	// Copy files
	files := []string{"context.md", "decisions.md", "continuation.md", "summary.txt"}
	for _, file := range files {
		srcFile := filepath.Join(fromPath, file)
		dstFile := filepath.Join(toPath, file)

		data, err := os.ReadFile(srcFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		if err := os.WriteFile(dstFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
	}

	// Copy research directory
	srcResearch := filepath.Join(fromPath, "research")
	dstResearch := filepath.Join(toPath, "research")

	entries, err := os.ReadDir(srcResearch)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read research directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcFile := filepath.Join(srcResearch, entry.Name())
		dstFile := filepath.Join(dstResearch, entry.Name())

		data, err := os.ReadFile(srcFile)
		if err != nil {
			return fmt.Errorf("failed to read research file %s: %w", entry.Name(), err)
		}

		if err := os.WriteFile(dstFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write research file %s: %w", entry.Name(), err)
		}
	}

	return nil
}
