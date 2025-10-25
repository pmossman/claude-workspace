package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a git repository for testing
func setupGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user (required for commits)
	configUser := exec.Command("git", "config", "user.name", "Test User")
	configUser.Dir = repoPath
	configUser.Run()

	configEmail := exec.Command("git", "config", "user.email", "test@example.com")
	configEmail.Dir = repoPath
	configEmail.Run()

	// Create initial commit
	readmePath := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repo"), 0644)
	require.NoError(t, err)

	addCmd := exec.Command("git", "add", "README.md")
	addCmd.Dir = repoPath
	err = addCmd.Run()
	require.NoError(t, err)

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = repoPath
	err = commitCmd.Run()
	require.NoError(t, err)

	return repoPath
}

func TestGetCurrentBranch(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Should be on main or master branch
	branch, err := GetCurrentBranch(repoPath)
	require.NoError(t, err)
	// Git defaults to either "master" or "main" depending on version
	assert.Contains(t, []string{"master", "main"}, branch)
}

func TestGetCurrentBranch_DifferentBranch(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Create and checkout a new branch
	cmd := exec.Command("git", "checkout", "-b", "feature-branch")
	cmd.Dir = repoPath
	err := cmd.Run()
	require.NoError(t, err)

	// Should return the new branch name
	branch, err := GetCurrentBranch(repoPath)
	require.NoError(t, err)
	assert.Equal(t, "feature-branch", branch)
}

func TestGetCurrentBranch_NonGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Should return error for non-git directory
	_, err := GetCurrentBranch(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current branch")
}

func TestGetCurrentBranch_NonExistentPath(t *testing.T) {
	// Should return error for non-existent path
	_, err := GetCurrentBranch("/nonexistent/path")
	assert.Error(t, err)
}

func TestIsGitRepo(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Should return true for git repo
	assert.True(t, IsGitRepo(repoPath))
}

func TestIsGitRepo_NonGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Should return false for non-git directory
	assert.False(t, IsGitRepo(tmpDir))
}

func TestIsGitRepo_NonExistentPath(t *testing.T) {
	// Should return false for non-existent path
	assert.False(t, IsGitRepo("/nonexistent/path"))
}

func TestIsGitRepo_SubDirectory(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Create subdirectory
	subDir := filepath.Join(repoPath, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Should still return true for subdirectory of git repo
	assert.True(t, IsGitRepo(subDir))
}

func TestGetRemoteURL(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Add a remote
	remoteURL := "https://github.com/test/repo.git"
	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = repoPath
	err := cmd.Run()
	require.NoError(t, err)

	// Should return the remote URL
	url, err := GetRemoteURL(repoPath)
	require.NoError(t, err)
	assert.Equal(t, remoteURL, url)
}

func TestGetRemoteURL_NoRemote(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Should return error when no remote exists
	_, err := GetRemoteURL(repoPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get remote URL")
}

func TestGetRemoteURL_NonGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Should return error for non-git directory
	_, err := GetRemoteURL(tmpDir)
	assert.Error(t, err)
}

func TestClone_LocalPath(t *testing.T) {
	// Create source repository
	sourceRepo := setupGitRepo(t)

	// Clone to new location
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "cloned-repo")

	err := Clone(sourceRepo, destPath)
	require.NoError(t, err)

	// Verify clone exists and is a git repo
	assert.DirExists(t, destPath)
	assert.True(t, IsGitRepo(destPath))

	// Verify README.md was cloned
	readmePath := filepath.Join(destPath, "README.md")
	assert.FileExists(t, readmePath)

	// Verify content
	content, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	assert.Equal(t, "# Test Repo", string(content))
}

func TestClone_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "cloned-repo")

	// Should return error for invalid URL
	err := Clone("https://invalid-git-url-that-does-not-exist.com/repo.git", destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repository")
}

func TestClone_ExistingDestination(t *testing.T) {
	sourceRepo := setupGitRepo(t)

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "cloned-repo")

	// Create destination directory with a file (non-empty)
	err := os.MkdirAll(destPath, 0755)
	require.NoError(t, err)

	// Add a file to make directory non-empty
	dummyFile := filepath.Join(destPath, "existing.txt")
	err = os.WriteFile(dummyFile, []byte("existing content"), 0644)
	require.NoError(t, err)

	// Should return error when destination already exists and is non-empty
	err = Clone(sourceRepo, destPath)
	assert.Error(t, err)
}

func TestClone_VerifyBranch(t *testing.T) {
	// Create source repository
	sourceRepo := setupGitRepo(t)

	// Clone to new location
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "cloned-repo")

	err := Clone(sourceRepo, destPath)
	require.NoError(t, err)

	// Verify we're on the default branch
	branch, err := GetCurrentBranch(destPath)
	require.NoError(t, err)
	assert.Contains(t, []string{"master", "main"}, branch)
}

func TestGetRemoteURL_SSHFormat(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Add SSH remote
	remoteURL := "git@github.com:test/repo.git"
	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = repoPath
	err := cmd.Run()
	require.NoError(t, err)

	// Should return the SSH remote URL
	url, err := GetRemoteURL(repoPath)
	require.NoError(t, err)
	assert.Equal(t, remoteURL, url)
}

func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Get current commit hash
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	require.NoError(t, err)
	commitHash := string(output[:7]) // Use short hash

	// Checkout specific commit (detached HEAD)
	checkoutCmd := exec.Command("git", "checkout", commitHash)
	checkoutCmd.Dir = repoPath
	err = checkoutCmd.Run()
	require.NoError(t, err)

	// Should return "HEAD" in detached state
	branch, err := GetCurrentBranch(repoPath)
	require.NoError(t, err)
	assert.Equal(t, "HEAD", branch)
}

func TestIsGitRepo_GitFileInsteadOfDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .git file (like git submodules or worktrees use)
	gitFile := filepath.Join(tmpDir, ".git")
	err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0644)
	require.NoError(t, err)

	// Git should still recognize this as a git repository
	// Note: This might return false since it's not a real submodule setup
	result := IsGitRepo(tmpDir)
	// Document the behavior: returns false for .git file without real repo
	assert.False(t, result)
}

func TestGetRemoteURL_MultipleRemotes(t *testing.T) {
	repoPath := setupGitRepo(t)

	// Add multiple remotes
	originURL := "https://github.com/test/origin.git"
	upstreamURL := "https://github.com/test/upstream.git"

	cmd1 := exec.Command("git", "remote", "add", "origin", originURL)
	cmd1.Dir = repoPath
	err := cmd1.Run()
	require.NoError(t, err)

	cmd2 := exec.Command("git", "remote", "add", "upstream", upstreamURL)
	cmd2.Dir = repoPath
	err = cmd2.Run()
	require.NoError(t, err)

	// Should return origin URL (function specifically gets origin)
	url, err := GetRemoteURL(repoPath)
	require.NoError(t, err)
	assert.Equal(t, originURL, url)
}
