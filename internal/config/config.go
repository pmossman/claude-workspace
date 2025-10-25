package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	StatusActive   = "active"
	StatusIdle     = "idle"
	StatusArchived = "archived"
)

type Remote struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	CloneBaseDir string `json:"clone_base_dir"`
}

type Clone struct {
	Path         string    `json:"path"`
	RemoteName   string    `json:"remote_name"`
	CreatedAt    time.Time `json:"created_at"`
	InUseBy      string    `json:"in_use_by,omitempty"` // workspace name, empty if free
	CurrentBranch string   `json:"current_branch,omitempty"`
}

type Workspace struct {
	Name       string    `json:"name"`
	RepoPath   string    `json:"repo_path"`            // deprecated, kept for backward compat
	ClonePath  string    `json:"clone_path,omitempty"` // new field
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
	Status     string    `json:"status"`
	SessionPID int       `json:"session_pid,omitempty"`
}

type Settings struct {
	WorkspaceDir      string `json:"workspace_dir"`
	AutoStartClaude   bool   `json:"auto_start_claude"`
	RequireSessionLock bool   `json:"require_session_lock"`
	ClaudeCommand     string `json:"claude_command"`
}

type Config struct {
	Workspaces map[string]*Workspace `json:"workspaces"`
	Remotes    map[string]*Remote    `json:"remotes"`
	Clones     map[string]*Clone     `json:"clones"` // keyed by path
	Settings   Settings              `json:"settings"`
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".claude-workspaces", "config.json"), nil
}

// Load reads the config from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return NewDefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Initialize maps if nil (backward compatibility)
	if cfg.Remotes == nil {
		cfg.Remotes = make(map[string]*Remote)
	}
	if cfg.Clones == nil {
		cfg.Clones = make(map[string]*Clone)
	}
	if cfg.Workspaces == nil {
		cfg.Workspaces = make(map[string]*Workspace)
	}

	return &cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// NewDefaultConfig returns a config with default settings
func NewDefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Workspaces: make(map[string]*Workspace),
		Remotes:    make(map[string]*Remote),
		Clones:     make(map[string]*Clone),
		Settings: Settings{
			WorkspaceDir:       filepath.Join(home, ".claude-workspaces"),
			AutoStartClaude:    true,
			RequireSessionLock: true,
			ClaudeCommand:      "claude",
		},
	}
}

// ValidateWorkspaceName checks if a workspace name is valid
// Valid names must:
// - Not be empty
// - Not contain spaces
// - Not contain path separators (/, \)
// - Not contain special shell characters that could cause issues
func ValidateWorkspaceName(name string) error {
	if name == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}

	// Check for spaces
	if strings.Contains(name, " ") {
		return fmt.Errorf("workspace name cannot contain spaces: '%s'", name)
	}

	// Check for path separators
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("workspace name cannot contain path separators: '%s'", name)
	}

	// Check for other problematic characters
	invalidChars := []string{":", "*", "?", "\"", "<", ">", "|", "\t", "\n"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("workspace name contains invalid character '%s': '%s'", char, name)
		}
	}

	// Check for path traversal patterns
	if strings.Contains(name, "..") {
		return fmt.Errorf("workspace name cannot contain '..': '%s'", name)
	}

	return nil
}

// AddWorkspace adds a new workspace to the config
func (c *Config) AddWorkspace(name, repoPath string) error {
	// Validate workspace name
	if err := ValidateWorkspaceName(name); err != nil {
		return err
	}

	if _, exists := c.Workspaces[name]; exists {
		return fmt.Errorf("workspace '%s' already exists", name)
	}

	c.Workspaces[name] = &Workspace{
		Name:       name,
		RepoPath:   repoPath,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Status:     StatusIdle,
	}

	return nil
}

// GetWorkspace retrieves a workspace by name
func (c *Config) GetWorkspace(name string) (*Workspace, error) {
	ws, exists := c.Workspaces[name]
	if !exists {
		return nil, fmt.Errorf("workspace '%s' not found", name)
	}
	return ws, nil
}

// UpdateWorkspaceStatus updates the status and last active time
func (c *Config) UpdateWorkspaceStatus(name, status string, pid int) error {
	ws, err := c.GetWorkspace(name)
	if err != nil {
		return err
	}

	ws.Status = status
	ws.LastActive = time.Now()
	ws.SessionPID = pid

	return nil
}

// RemoveWorkspace removes a workspace from the config
func (c *Config) RemoveWorkspace(name string) error {
	if _, exists := c.Workspaces[name]; !exists {
		return fmt.Errorf("workspace '%s' not found", name)
	}
	delete(c.Workspaces, name)
	return nil
}

// GetRepoPath returns the repository path for a workspace (handles both old and new formats)
func (w *Workspace) GetRepoPath() string {
	if w.ClonePath != "" {
		return w.ClonePath
	}
	return w.RepoPath
}

// Remote management

// AddRemote adds a new remote to the config
func (c *Config) AddRemote(name, url, cloneBaseDir string) error {
	if _, exists := c.Remotes[name]; exists {
		return fmt.Errorf("remote '%s' already exists", name)
	}

	c.Remotes[name] = &Remote{
		Name:         name,
		URL:          url,
		CloneBaseDir: cloneBaseDir,
	}

	return nil
}

// GetRemote retrieves a remote by name
func (c *Config) GetRemote(name string) (*Remote, error) {
	remote, exists := c.Remotes[name]
	if !exists {
		return nil, fmt.Errorf("remote '%s' not found", name)
	}
	return remote, nil
}

// Clone management

// AddClone adds a new clone to the config
func (c *Config) AddClone(path, remoteName string) error {
	if _, exists := c.Clones[path]; exists {
		return fmt.Errorf("clone at '%s' already exists", path)
	}

	c.Clones[path] = &Clone{
		Path:       path,
		RemoteName: remoteName,
		CreatedAt:  time.Now(),
		InUseBy:    "",
	}

	return nil
}

// GetClone retrieves a clone by path
func (c *Config) GetClone(path string) (*Clone, error) {
	clone, exists := c.Clones[path]
	if !exists {
		return nil, fmt.Errorf("clone at '%s' not found", path)
	}
	return clone, nil
}

// GetClonesForRemote returns all clones for a given remote
func (c *Config) GetClonesForRemote(remoteName string) []*Clone {
	var clones []*Clone
	for _, clone := range c.Clones {
		if clone.RemoteName == remoteName {
			clones = append(clones, clone)
		}
	}
	return clones
}

// FindFreeClone finds an available (not in use) clone for a remote
func (c *Config) FindFreeClone(remoteName string) *Clone {
	for _, clone := range c.Clones {
		if clone.RemoteName == remoteName && clone.InUseBy == "" {
			return clone
		}
	}
	return nil
}

// FindIdleClones finds clones that are in use by idle workspaces
func (c *Config) FindIdleClones(remoteName string) []*Clone {
	var idleClones []*Clone
	for _, clone := range c.Clones {
		if clone.RemoteName == remoteName && clone.InUseBy != "" {
			// Check if the workspace is idle
			if ws, err := c.GetWorkspace(clone.InUseBy); err == nil && ws.Status == StatusIdle {
				idleClones = append(idleClones, clone)
			}
		}
	}
	return idleClones
}

// AssignCloneToWorkspace marks a clone as in use by a workspace
func (c *Config) AssignCloneToWorkspace(clonePath, workspaceName string) error {
	clone, err := c.GetClone(clonePath)
	if err != nil {
		return err
	}

	if clone.InUseBy != "" && clone.InUseBy != workspaceName {
		return fmt.Errorf("clone is already in use by workspace '%s'", clone.InUseBy)
	}

	clone.InUseBy = workspaceName
	return nil
}

// FreeClone marks a clone as available (not in use)
func (c *Config) FreeClone(clonePath string) error {
	clone, err := c.GetClone(clonePath)
	if err != nil {
		return err
	}

	clone.InUseBy = ""
	return nil
}

// GetNextCloneNumber returns the next available clone number for a remote
func (c *Config) GetNextCloneNumber(remoteName string) int {
	maxNum := 0
	for _, clone := range c.Clones {
		if clone.RemoteName == remoteName {
			// Extract number from path
			base := filepath.Base(clone.Path)
			var num int
			if _, err := fmt.Sscanf(base, "%d", &num); err == nil {
				if num > maxNum {
					maxNum = num
				}
			}
		}
	}
	return maxNum + 1
}
