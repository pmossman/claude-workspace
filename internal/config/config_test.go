package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create temp dir for tests
func setupTestDir(t *testing.T) string {
	tmpDir := t.TempDir()
	return tmpDir
}

// Helper to create test config
func createTestConfig(t *testing.T, tmpDir string) *Config {
	cfg := NewDefaultConfig()
	cfg.Settings.WorkspaceDir = tmpDir
	return cfg
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Workspaces)
	assert.NotNil(t, cfg.Remotes)
	assert.NotNil(t, cfg.Clones)
	assert.NotNil(t, cfg.Settings)

	// Check default settings
	assert.True(t, cfg.Settings.AutoStartClaude)
	assert.True(t, cfg.Settings.RequireSessionLock)
	assert.Equal(t, "claude", cfg.Settings.ClaudeCommand)
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Override config path for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config
	cfg := NewDefaultConfig()
	cfg.Settings.WorkspaceDir = filepath.Join(tmpDir, ".claude-workspaces")

	// Add test data
	cfg.Workspaces["test-ws"] = &Workspace{
		Name:       "test-ws",
		RepoPath:   "/tmp/test-repo",
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Status:     StatusIdle,
	}

	// Save
	err := cfg.Save()
	require.NoError(t, err)

	// Verify file exists
	configPath, err := GetConfigPath()
	require.NoError(t, err)
	assert.FileExists(t, configPath)

	// Load
	loaded, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, loaded)

	// Verify data
	ws, err := loaded.GetWorkspace("test-ws")
	require.NoError(t, err)
	assert.Equal(t, "test-ws", ws.Name)
	assert.Equal(t, "/tmp/test-repo", ws.RepoPath)
	assert.Equal(t, StatusIdle, ws.Status)
}

func TestConfig_SaveCreatesDirectory(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Override config path
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := NewDefaultConfig()
	cfg.Settings.WorkspaceDir = filepath.Join(tmpDir, ".claude-workspaces")

	// Save should create directory
	err := cfg.Save()
	require.NoError(t, err)

	// Verify directory was created
	configPath, err := GetConfigPath()
	require.NoError(t, err)
	configDir := filepath.Dir(configPath)
	assert.DirExists(t, configDir)
}

func TestConfig_LoadNonExistent(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Override config path
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Load should return default config if file doesn't exist
	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have default settings
	assert.True(t, cfg.Settings.AutoStartClaude)
	assert.Equal(t, "claude", cfg.Settings.ClaudeCommand)
}

func TestConfig_LoadInvalidJSON(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Override config path
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create invalid JSON file
	configPath, err := GetConfigPath()
	require.NoError(t, err)
	os.MkdirAll(filepath.Dir(configPath), 0755)
	err = os.WriteFile(configPath, []byte("invalid json{"), 0644)
	require.NoError(t, err)

	// Load should fail
	_, err = Load()
	assert.Error(t, err)
}

func TestConfig_AddWorkspace(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	err := cfg.AddWorkspace("test-ws", "/tmp/test-repo")
	require.NoError(t, err)

	// Verify workspace was added
	ws, err := cfg.GetWorkspace("test-ws")
	require.NoError(t, err)
	assert.Equal(t, "test-ws", ws.Name)
	assert.Equal(t, "/tmp/test-repo", ws.RepoPath)
	assert.Equal(t, StatusIdle, ws.Status)
	assert.False(t, ws.CreatedAt.IsZero())
	assert.False(t, ws.LastActive.IsZero())
}

func TestConfig_AddWorkspace_Duplicate(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add first time
	err := cfg.AddWorkspace("test-ws", "/tmp/test-repo")
	require.NoError(t, err)

	// Add second time should fail
	err = cfg.AddWorkspace("test-ws", "/tmp/test-repo-2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestConfig_AddWorkspace_EmptyName(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	err := cfg.AddWorkspace("", "/tmp/test-repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestValidateWorkspaceName(t *testing.T) {
	tests := []struct {
		name      string
		wsName    string
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid simple name",
			wsName:    "myworkspace",
			wantError: false,
		},
		{
			name:      "valid name with dash",
			wsName:    "my-workspace",
			wantError: false,
		},
		{
			name:      "valid name with underscore",
			wsName:    "my_workspace",
			wantError: false,
		},
		{
			name:      "valid name with number",
			wsName:    "workspace123",
			wantError: false,
		},
		{
			name:      "empty name",
			wsName:    "",
			wantError: true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "name with space",
			wsName:    "my workspace",
			wantError: true,
			errMsg:    "cannot contain spaces",
		},
		{
			name:      "name with multiple spaces",
			wsName:    "my   workspace",
			wantError: true,
			errMsg:    "cannot contain spaces",
		},
		{
			name:      "name with forward slash",
			wsName:    "my/workspace",
			wantError: true,
			errMsg:    "cannot contain path separators",
		},
		{
			name:      "name with backslash",
			wsName:    "my\\workspace",
			wantError: true,
			errMsg:    "cannot contain path separators",
		},
		{
			name:      "name with colon",
			wsName:    "my:workspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with asterisk",
			wsName:    "my*workspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with question mark",
			wsName:    "my?workspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with quotes",
			wsName:    "my\"workspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with less than",
			wsName:    "my<workspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with greater than",
			wsName:    "my>workspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with pipe",
			wsName:    "my|workspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with tab",
			wsName:    "my\tworkspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with newline",
			wsName:    "my\nworkspace",
			wantError: true,
			errMsg:    "invalid character",
		},
		{
			name:      "name with path traversal",
			wsName:    "../escape",
			wantError: true,
			errMsg:    "cannot contain path separators", // Caught by slash check first
		},
		{
			name:      "name with double dot in middle",
			wsName:    "my..workspace",
			wantError: true,
			errMsg:    "cannot contain '..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspaceName(tt.wsName)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_AddWorkspace_InvalidNames(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Test that AddWorkspace rejects invalid names
	invalidNames := []string{
		"my workspace",
		"my/workspace",
		"my\\workspace",
		"my:workspace",
		"../escape",
	}

	for _, name := range invalidNames {
		err := cfg.AddWorkspace(name, "/tmp/test-repo")
		assert.Error(t, err, "should reject workspace name: %s", name)
	}
}

func TestConfig_GetWorkspace(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add workspace
	cfg.AddWorkspace("test-ws", "/tmp/test-repo")

	// Get existing workspace
	ws, err := cfg.GetWorkspace("test-ws")
	require.NoError(t, err)
	assert.Equal(t, "test-ws", ws.Name)

	// Get non-existent workspace
	_, err = cfg.GetWorkspace("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfig_RemoveWorkspace(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add workspace
	cfg.AddWorkspace("test-ws", "/tmp/test-repo")

	// Remove workspace
	err := cfg.RemoveWorkspace("test-ws")
	require.NoError(t, err)

	// Verify workspace was removed
	_, err = cfg.GetWorkspace("test-ws")
	assert.Error(t, err)
}

func TestConfig_UpdateWorkspaceStatus(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add workspace
	cfg.AddWorkspace("test-ws", "/tmp/test-repo")

	// Update status
	err := cfg.UpdateWorkspaceStatus("test-ws", StatusActive, 1234)
	require.NoError(t, err)

	// Verify status was updated
	ws, _ := cfg.GetWorkspace("test-ws")
	assert.Equal(t, StatusActive, ws.Status)
	assert.Equal(t, 1234, ws.SessionPID)

	// Update to idle
	err = cfg.UpdateWorkspaceStatus("test-ws", StatusIdle, 0)
	require.NoError(t, err)

	ws, _ = cfg.GetWorkspace("test-ws")
	assert.Equal(t, StatusIdle, ws.Status)
	assert.Equal(t, 0, ws.SessionPID)
}

func TestWorkspace_GetRepoPath(t *testing.T) {
	tests := []struct {
		name      string
		workspace Workspace
		expected  string
	}{
		{
			name: "ClonePath takes precedence",
			workspace: Workspace{
				ClonePath: "/new/path",
				RepoPath:  "/old/path",
			},
			expected: "/new/path",
		},
		{
			name: "Falls back to RepoPath",
			workspace: Workspace{
				ClonePath: "",
				RepoPath:  "/old/path",
			},
			expected: "/old/path",
		},
		{
			name: "Only ClonePath set",
			workspace: Workspace{
				ClonePath: "/only/clone",
			},
			expected: "/only/clone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.workspace.GetRepoPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_AddRemote(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	err := cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")
	require.NoError(t, err)

	// Verify remote was added
	remote, err := cfg.GetRemote("origin")
	require.NoError(t, err)
	assert.Equal(t, "origin", remote.Name)
	assert.Equal(t, "git@github.com:user/repo.git", remote.URL)
	assert.Equal(t, "/tmp/clones", remote.CloneBaseDir)
}

func TestConfig_AddRemote_Duplicate(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add first time
	err := cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")
	require.NoError(t, err)

	// Add second time should fail
	err = cfg.AddRemote("origin", "git@github.com:user/repo2.git", "/tmp/clones2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestConfig_GetRemote(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")

	// Get existing remote
	remote, err := cfg.GetRemote("origin")
	require.NoError(t, err)
	assert.Equal(t, "origin", remote.Name)

	// Get non-existent remote
	_, err = cfg.GetRemote("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfig_AddClone(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote first
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")

	// Add clone
	err := cfg.AddClone("/tmp/clones/1", "origin")
	require.NoError(t, err)

	// Verify clone was added
	clone, err := cfg.GetClone("/tmp/clones/1")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/clones/1", clone.Path)
	assert.Equal(t, "origin", clone.RemoteName)
	assert.Equal(t, "", clone.InUseBy)
	assert.False(t, clone.CreatedAt.IsZero())
}

func TestConfig_GetClonesForRemote(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")

	// Add clones
	cfg.AddClone("/tmp/clones/1", "origin")
	cfg.AddClone("/tmp/clones/2", "origin")

	// Get clones
	clones := cfg.GetClonesForRemote("origin")
	assert.Len(t, clones, 2)

	// Check they're sorted by path
	assert.Equal(t, "/tmp/clones/1", clones[0].Path)
	assert.Equal(t, "/tmp/clones/2", clones[1].Path)
}

func TestConfig_FindFreeClone(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")

	// Add clones
	cfg.AddClone("/tmp/clones/1", "origin")
	cfg.AddClone("/tmp/clones/2", "origin")

	// Assign first clone
	cfg.AssignCloneToWorkspace("/tmp/clones/1", "test-ws")

	// Find free clone (should return clone 2)
	freeClone := cfg.FindFreeClone("origin")
	require.NotNil(t, freeClone)
	assert.Equal(t, "/tmp/clones/2", freeClone.Path)

	// Assign second clone
	cfg.AssignCloneToWorkspace("/tmp/clones/2", "test-ws-2")

	// Find free clone (should return nil)
	freeClone = cfg.FindFreeClone("origin")
	assert.Nil(t, freeClone)
}

func TestConfig_AssignCloneToWorkspace(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote and clone
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")
	cfg.AddClone("/tmp/clones/1", "origin")

	// Assign clone
	err := cfg.AssignCloneToWorkspace("/tmp/clones/1", "test-ws")
	require.NoError(t, err)

	// Verify assignment
	clone, _ := cfg.GetClone("/tmp/clones/1")
	assert.Equal(t, "test-ws", clone.InUseBy)
}

func TestConfig_FreeClone(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote and clone
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")
	cfg.AddClone("/tmp/clones/1", "origin")

	// Assign clone
	cfg.AssignCloneToWorkspace("/tmp/clones/1", "test-ws")

	// Free clone
	err := cfg.FreeClone("/tmp/clones/1")
	require.NoError(t, err)

	// Verify it's free
	clone, _ := cfg.GetClone("/tmp/clones/1")
	assert.Equal(t, "", clone.InUseBy)
}

func TestConfig_GetNextCloneNumber(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")

	// Should start at 1
	num := cfg.GetNextCloneNumber("origin")
	assert.Equal(t, 1, num)

	// Add clone 1
	cfg.AddClone("/tmp/clones/1", "origin")

	// Should return 2
	num = cfg.GetNextCloneNumber("origin")
	assert.Equal(t, 2, num)

	// Add clone 3 (skipping 2)
	cfg.AddClone("/tmp/clones/3", "origin")

	// Should return 4 (doesn't fill gaps, returns max+1)
	num = cfg.GetNextCloneNumber("origin")
	assert.Equal(t, 4, num)
}

func TestConfig_FindIdleClones(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add remote
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")

	// Add workspaces
	cfg.AddWorkspace("active-ws", "/tmp/clones/1")
	cfg.AddWorkspace("idle-ws", "/tmp/clones/2")

	// Add clones
	cfg.AddClone("/tmp/clones/1", "origin")
	cfg.AddClone("/tmp/clones/2", "origin")
	cfg.AddClone("/tmp/clones/3", "origin")

	// Assign clones to workspaces
	cfg.AssignCloneToWorkspace("/tmp/clones/1", "active-ws")
	cfg.AssignCloneToWorkspace("/tmp/clones/2", "idle-ws")
	// Clone 3 is free

	// Set workspace statuses
	cfg.UpdateWorkspaceStatus("active-ws", StatusActive, 1234)
	cfg.UpdateWorkspaceStatus("idle-ws", StatusIdle, 0)

	// Find idle clones
	idleClones := cfg.FindIdleClones("origin")
	assert.Len(t, idleClones, 1)
	assert.Equal(t, "/tmp/clones/2", idleClones[0].Path)
}

func TestConfig_JSONRoundTrip(t *testing.T) {
	cfg := createTestConfig(t, setupTestDir(t))

	// Add test data
	cfg.AddWorkspace("test-ws", "/tmp/repo")
	cfg.AddRemote("origin", "git@github.com:user/repo.git", "/tmp/clones")
	cfg.AddClone("/tmp/clones/1", "origin")
	cfg.AssignCloneToWorkspace("/tmp/clones/1", "test-ws")

	// Marshal to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)

	// Unmarshal from JSON
	var loaded Config
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	// Verify data
	assert.Len(t, loaded.Workspaces, 1)
	assert.Len(t, loaded.Remotes, 1)
	assert.Len(t, loaded.Clones, 1)

	ws, _ := loaded.GetWorkspace("test-ws")
	assert.Equal(t, "test-ws", ws.Name)

	clone, _ := loaded.GetClone("/tmp/clones/1")
	assert.Equal(t, "test-ws", clone.InUseBy)
}
