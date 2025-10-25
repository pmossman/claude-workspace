package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	baseDir := "/tmp/test-workspaces"
	mgr := NewManager(baseDir)

	assert.NotNil(t, mgr)
	assert.Equal(t, baseDir, mgr.baseDir)
}

func TestManager_GetPath(t *testing.T) {
	mgr := NewManager("/tmp/workspaces")

	path := mgr.GetPath("test-ws")
	assert.Equal(t, "/tmp/workspaces/test-ws", path)
}

func TestManager_Create(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.Create("test-ws")
	require.NoError(t, err)

	// Verify workspace directory exists
	wsPath := mgr.GetPath("test-ws")
	assert.DirExists(t, wsPath)

	// Verify all files exist
	files := []string{"context.md", "decisions.md", "continuation.md", "summary.txt"}
	for _, file := range files {
		filePath := filepath.Join(wsPath, file)
		assert.FileExists(t, filePath)

		// Verify files are empty initially
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Empty(t, data)
	}

	// Verify research directory exists
	researchPath := filepath.Join(wsPath, "research")
	assert.DirExists(t, researchPath)
}

func TestManager_Create_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create first time
	err := mgr.Create("test-ws")
	require.NoError(t, err)

	// Create second time should succeed (mkdir -p)
	// Note: Current implementation doesn't check if workspace already exists
	err = mgr.Create("test-ws")
	require.NoError(t, err)
}

func TestManager_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Should not exist initially
	assert.False(t, mgr.Exists("test-ws"))

	// Create workspace
	mgr.Create("test-ws")

	// Should exist now
	assert.True(t, mgr.Exists("test-ws"))

	// Non-existent workspace
	assert.False(t, mgr.Exists("nonexistent"))
}

func TestManager_GetSummary(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// Empty summary should return "(no summary)"
	summary := mgr.GetSummary("test-ws")
	assert.Equal(t, "(no summary)", summary)

	// Write summary
	summaryPath := filepath.Join(mgr.GetPath("test-ws"), "summary.txt")
	err := os.WriteFile(summaryPath, []byte("Test workspace summary"), 0644)
	require.NoError(t, err)

	// Should return summary
	summary = mgr.GetSummary("test-ws")
	assert.Equal(t, "Test workspace summary", summary)

	// Summary with whitespace should be trimmed
	err = os.WriteFile(summaryPath, []byte("  Whitespace summary  \n"), 0644)
	require.NoError(t, err)

	summary = mgr.GetSummary("test-ws")
	assert.Equal(t, "Whitespace summary", summary)
}

func TestManager_GetSummary_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Non-existent workspace should return "(no summary)"
	summary := mgr.GetSummary("nonexistent")
	assert.Equal(t, "(no summary)", summary)
}

func TestManager_GetContinuation(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// Empty continuation should return ""
	continuation := mgr.GetContinuation("test-ws")
	assert.Empty(t, continuation)

	// Write continuation
	contPath := filepath.Join(mgr.GetPath("test-ws"), "continuation.md")
	content := "# Continuation\n\nThis is the continuation prompt."
	err := os.WriteFile(contPath, []byte(content), 0644)
	require.NoError(t, err)

	// Should return continuation
	continuation = mgr.GetContinuation("test-ws")
	assert.Equal(t, content, continuation)
}

func TestManager_GetContext(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// Empty context should return "(no context yet)"
	context := mgr.GetContext("test-ws")
	assert.Equal(t, "(no context yet)", context)

	// Write short context
	contextPath := filepath.Join(mgr.GetPath("test-ws"), "context.md")
	shortContent := "Short context"
	err := os.WriteFile(contextPath, []byte(shortContent), 0644)
	require.NoError(t, err)

	context = mgr.GetContext("test-ws")
	assert.Equal(t, shortContent, context)

	// Write long context (> 200 chars)
	longContent := make([]byte, 300)
	for i := range longContent {
		longContent[i] = 'a'
	}
	err = os.WriteFile(contextPath, longContent, 0644)
	require.NoError(t, err)

	context = mgr.GetContext("test-ws")
	assert.Len(t, context, 203) // 200 chars + "..."
	assert.True(t, len(context) <= 203)
	assert.Contains(t, context, "...")
}

func TestManager_CreateLock(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// Create lock
	err := mgr.CreateLock("test-ws", 12345)
	require.NoError(t, err)

	// Verify lock file exists
	lockPath := filepath.Join(mgr.GetPath("test-ws"), ".lock")
	assert.FileExists(t, lockPath)

	// Verify PID is written
	data, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Equal(t, "12345", string(data))
}

func TestManager_RemoveLock(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// Create lock
	mgr.CreateLock("test-ws", 12345)

	// Remove lock
	err := mgr.RemoveLock("test-ws")
	require.NoError(t, err)

	// Verify lock file doesn't exist
	lockPath := filepath.Join(mgr.GetPath("test-ws"), ".lock")
	assert.NoFileExists(t, lockPath)

	// Removing non-existent lock should not error
	err = mgr.RemoveLock("test-ws")
	require.NoError(t, err)
}

func TestManager_CheckLock(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// No lock should return false
	locked, pid, err := mgr.CheckLock("test-ws")
	require.NoError(t, err)
	assert.False(t, locked)
	assert.Equal(t, 0, pid)

	// Create lock with current process PID (should be running)
	currentPID := os.Getpid()
	mgr.CreateLock("test-ws", currentPID)

	// Should be locked
	locked, pid, err = mgr.CheckLock("test-ws")
	require.NoError(t, err)
	assert.True(t, locked)
	assert.Equal(t, currentPID, pid)

	// Create lock with impossible PID (very high number, likely not running)
	mgr.CreateLock("test-ws", 999999)

	// Should not be locked (process doesn't exist)
	locked, pid, err = mgr.CheckLock("test-ws")
	require.NoError(t, err)
	assert.False(t, locked)
	assert.Equal(t, 999999, pid)
}

func TestManager_CheckLock_InvalidPID(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// Create invalid lock file
	lockPath := filepath.Join(mgr.GetPath("test-ws"), ".lock")
	err := os.WriteFile(lockPath, []byte("not-a-number"), 0644)
	require.NoError(t, err)

	// Should return error
	_, _, err = mgr.CheckLock("test-ws")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid lock file")
}

func TestManager_Archive(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create workspace
	mgr.Create("test-ws")

	// Write some content
	summaryPath := filepath.Join(mgr.GetPath("test-ws"), "summary.txt")
	err := os.WriteFile(summaryPath, []byte("Test summary"), 0644)
	require.NoError(t, err)

	// Archive workspace
	err = mgr.Archive("test-ws")
	require.NoError(t, err)

	// Original workspace should not exist
	assert.False(t, mgr.Exists("test-ws"))

	// Archived workspace should exist
	archivedPath := filepath.Join(tmpDir, "archived", "test-ws")
	assert.DirExists(t, archivedPath)

	// Content should be preserved
	archivedSummary := filepath.Join(archivedPath, "summary.txt")
	data, err := os.ReadFile(archivedSummary)
	require.NoError(t, err)
	assert.Equal(t, "Test summary", string(data))
}

func TestManager_Archive_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Archive non-existent workspace
	err := mgr.Archive("nonexistent")
	assert.Error(t, err)
}

func TestManager_Clone(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create source workspace
	mgr.Create("source-ws")

	// Write content to source
	sourcePath := mgr.GetPath("source-ws")
	os.WriteFile(filepath.Join(sourcePath, "summary.txt"), []byte("Source summary"), 0644)
	os.WriteFile(filepath.Join(sourcePath, "context.md"), []byte("Source context"), 0644)
	os.WriteFile(filepath.Join(sourcePath, "decisions.md"), []byte("Source decisions"), 0644)
	os.WriteFile(filepath.Join(sourcePath, "continuation.md"), []byte("Source continuation"), 0644)

	// Write research file
	researchFile := filepath.Join(sourcePath, "research", "findings.md")
	os.WriteFile(researchFile, []byte("Research findings"), 0644)

	// Clone workspace
	err := mgr.Clone("source-ws", "cloned-ws")
	require.NoError(t, err)

	// Verify cloned workspace exists
	assert.True(t, mgr.Exists("cloned-ws"))

	// Verify all files were copied
	clonedPath := mgr.GetPath("cloned-ws")

	files := map[string]string{
		"summary.txt":      "Source summary",
		"context.md":       "Source context",
		"decisions.md":     "Source decisions",
		"continuation.md":  "Source continuation",
		"research/findings.md": "Research findings",
	}

	for file, expectedContent := range files {
		filePath := filepath.Join(clonedPath, file)
		assert.FileExists(t, filePath)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(data))
	}
}

func TestManager_Clone_SourceNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Clone from non-existent source
	err := mgr.Clone("nonexistent", "dest")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestManager_Clone_DestAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create both workspaces
	mgr.Create("source-ws")
	mgr.Create("dest-ws")

	// Clone should fail
	err := mgr.Clone("source-ws", "dest-ws")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_Clone_EmptyResearch(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create source workspace (research dir exists but is empty)
	mgr.Create("source-ws")

	// Clone should succeed even with empty research dir
	err := mgr.Clone("source-ws", "cloned-ws")
	require.NoError(t, err)

	// Verify cloned workspace has research dir
	clonedResearch := filepath.Join(mgr.GetPath("cloned-ws"), "research")
	assert.DirExists(t, clonedResearch)
}

func TestManager_Clone_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create source workspace
	mgr.Create("source-ws")

	// Delete a file
	os.Remove(filepath.Join(mgr.GetPath("source-ws"), "context.md"))

	// Clone should still succeed (skips missing files)
	err := mgr.Clone("source-ws", "cloned-ws")
	require.NoError(t, err)

	// Verify cloned workspace exists but missing file is still missing
	assert.True(t, mgr.Exists("cloned-ws"))

	// The file should exist but be empty (created by Create)
	clonedContext := filepath.Join(mgr.GetPath("cloned-ws"), "context.md")
	assert.FileExists(t, clonedContext)
}

func TestManager_GetPath_RelativeName(t *testing.T) {
	mgr := NewManager("/tmp/workspaces")

	// Test with path traversal attempt - should be prevented
	path := mgr.GetPath("../escape")
	// Should sanitize to just "escape" within baseDir (prevents traversal)
	assert.Equal(t, "/tmp/workspaces/escape", path)
	assert.Contains(t, path, "/tmp/workspaces/")

	// Test with multiple traversal attempts
	path = mgr.GetPath("../../etc/passwd")
	assert.Equal(t, "/tmp/workspaces/passwd", path)

	// Test with absolute path attempt
	path = mgr.GetPath("/etc/passwd")
	assert.Equal(t, "/tmp/workspaces/passwd", path)
}

func TestManager_Create_PermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// Create a directory with no write permissions
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0444)
	require.NoError(t, err)
	defer os.Chmod(readOnlyDir, 0755) // Cleanup

	mgr := NewManager(readOnlyDir)

	// Create should fail due to permissions
	err = mgr.Create("test-ws")
	assert.Error(t, err)
}
