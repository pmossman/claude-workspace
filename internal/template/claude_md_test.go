package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateClaudeMd(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	workspaceDir := filepath.Join(tmpDir, "workspace")
	err = os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err)

	// Generate CLAUDE.md
	err = GenerateClaudeMd("test-workspace", workspaceDir, repoPath)
	require.NoError(t, err)

	// Verify .claude directory was created
	claudeDir := filepath.Join(repoPath, ".claude")
	assert.DirExists(t, claudeDir)

	// Verify CLAUDE.md file was created
	claudeMd := filepath.Join(claudeDir, "CLAUDE.md")
	assert.FileExists(t, claudeMd)

	// Read and verify content
	content, err := os.ReadFile(claudeMd)
	require.NoError(t, err)

	contentStr := string(content)

	// Should contain workspace name
	assert.Contains(t, contentStr, "test-workspace")

	// Should contain workspace directory paths
	assert.Contains(t, contentStr, workspaceDir)

	// Should contain key instruction sections
	assert.Contains(t, contentStr, "context.md")
	assert.Contains(t, contentStr, "decisions.md")
	assert.Contains(t, contentStr, "continuation.md")
	assert.Contains(t, contentStr, "summary.txt")
	assert.Contains(t, contentStr, "research/")

	// Should contain workflow instructions (case-insensitive)
	contentLower := strings.ToLower(contentStr)
	assert.Contains(t, contentLower, "startup")
	assert.Contains(t, contentLower, "corrections")
	assert.Contains(t, contentLower, "research")
}

func TestGenerateClaudeMd_CreatesClaudeDir(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	workspaceDir := filepath.Join(tmpDir, "workspace")
	err = os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err)

	// .claude directory should not exist yet
	claudeDir := filepath.Join(repoPath, ".claude")
	assert.NoDirExists(t, claudeDir)

	// Generate should create it
	err = GenerateClaudeMd("test-workspace", workspaceDir, repoPath)
	require.NoError(t, err)

	assert.DirExists(t, claudeDir)
}

func TestGenerateClaudeMd_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	workspaceDir := filepath.Join(tmpDir, "workspace")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err)

	// Generate first time
	err = GenerateClaudeMd("first-workspace", workspaceDir, repoPath)
	require.NoError(t, err)

	// Generate again with different workspace name
	err = GenerateClaudeMd("second-workspace", workspaceDir, repoPath)
	require.NoError(t, err)

	// Should contain second workspace name
	claudeMd := filepath.Join(repoPath, ".claude", "CLAUDE.md")
	content, err := os.ReadFile(claudeMd)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "second-workspace")
	assert.NotContains(t, contentStr, "first-workspace")
}

func TestGenerateClaudeMd_InvalidRepoPath(t *testing.T) {
	workspaceDir := t.TempDir()

	// Generate with non-existent repo path
	err := GenerateClaudeMd("test-workspace", workspaceDir, "/nonexistent/path")
	assert.Error(t, err)
}

func TestEnsureGitignore_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	// .gitignore should not exist
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	assert.NoFileExists(t, gitignorePath)

	// Ensure should create it
	err = EnsureGitignore(repoPath)
	require.NoError(t, err)

	// Should exist now
	assert.FileExists(t, gitignorePath)

	// Should contain .claude/
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), ".claude/")
}

func TestEnsureGitignore_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	gitignorePath := filepath.Join(repoPath, ".gitignore")

	// Create existing .gitignore with .claude/
	existingContent := "# Existing content\n.claude/\n*.log\n"
	err = os.WriteFile(gitignorePath, []byte(existingContent), 0644)
	require.NoError(t, err)

	// Ensure should not modify it
	err = EnsureGitignore(repoPath)
	require.NoError(t, err)

	// Content should be unchanged
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(content))
}

func TestEnsureGitignore_AppendsIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	gitignorePath := filepath.Join(repoPath, ".gitignore")

	// Create existing .gitignore WITHOUT .claude/
	existingContent := "# Existing content\n*.log\nnode_modules/\n"
	err = os.WriteFile(gitignorePath, []byte(existingContent), 0644)
	require.NoError(t, err)

	// Ensure should append .claude/
	err = EnsureGitignore(repoPath)
	require.NoError(t, err)

	// Should contain both old and new content
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "*.log")
	assert.Contains(t, contentStr, "node_modules/")
	assert.Contains(t, contentStr, ".claude/")
}

func TestEnsureGitignore_NotFalsePositive(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	gitignorePath := filepath.Join(repoPath, ".gitignore")

	// Create .gitignore with "myclaude/" (contains ".claude/" as substring)
	existingContent := "myclaude/\n"
	err = os.WriteFile(gitignorePath, []byte(existingContent), 0644)
	require.NoError(t, err)

	// Implementation correctly adds .claude/ even though "myclaude/" contains ".claude/" as substring
	// This shows line-by-line checking works correctly
	err = EnsureGitignore(repoPath)
	require.NoError(t, err)

	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)

	// Both entries should be present
	contentStr := string(content)
	assert.Contains(t, contentStr, "myclaude/")
	assert.Contains(t, contentStr, ".claude/")
}

func TestEnsureGitignore_InvalidPath(t *testing.T) {
	// Ensure with non-existent repo path
	err := EnsureGitignore("/nonexistent/path")
	// Should not error, just skip
	// Current implementation will return error from Open
	assert.Error(t, err)
}

func TestRemoveClaudeMd(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	claudeDir := filepath.Join(repoPath, ".claude")
	err := os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)

	// Create CLAUDE.md
	claudeMd := filepath.Join(claudeDir, "CLAUDE.md")
	err = os.WriteFile(claudeMd, []byte("test content"), 0644)
	require.NoError(t, err)

	// Remove should delete it
	err = RemoveClaudeMd(repoPath)
	require.NoError(t, err)

	// File should not exist
	assert.NoFileExists(t, claudeMd)
}

func TestRemoveClaudeMd_AlreadyRemoved(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	// Remove non-existent file should not error
	err = RemoveClaudeMd(repoPath)
	require.NoError(t, err)
}

func TestRemoveClaudeMd_KeepsOtherFiles(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	claudeDir := filepath.Join(repoPath, ".claude")
	err := os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)

	// Create multiple files
	claudeMd := filepath.Join(claudeDir, "CLAUDE.md")
	otherFile := filepath.Join(claudeDir, "other.txt")
	err = os.WriteFile(claudeMd, []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(otherFile, []byte("keep this"), 0644)
	require.NoError(t, err)

	// Remove should only delete CLAUDE.md
	err = RemoveClaudeMd(repoPath)
	require.NoError(t, err)

	assert.NoFileExists(t, claudeMd)
	assert.FileExists(t, otherFile)
}

func TestTemplate_WorkspacePathFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	// Workspace dir with special path
	workspaceDir := filepath.Join(tmpDir, "my workspace", "test")
	err = os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err)

	err = GenerateClaudeMd("test", workspaceDir, repoPath)
	require.NoError(t, err)

	// Verify path is correctly embedded
	claudeMd := filepath.Join(repoPath, ".claude", "CLAUDE.md")
	content, err := os.ReadFile(claudeMd)
	require.NoError(t, err)

	// Should contain the workspace directory path
	assert.Contains(t, string(content), workspaceDir)
}

func TestTemplate_ContainsAllInstructions(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	workspaceDir := filepath.Join(tmpDir, "workspace")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err)

	err = GenerateClaudeMd("test", workspaceDir, repoPath)
	require.NoError(t, err)

	claudeMd := filepath.Join(repoPath, ".claude", "CLAUDE.md")
	content, err := os.ReadFile(claudeMd)
	require.NoError(t, err)

	contentStr := strings.ToLower(string(content))

	// Key workflow instructions that should be present
	keywords := []string{
		"startup",
		"during work",
		"corrections",
		"research",
		"every 30 min",
		"context.md",
		"decisions.md",
		"continuation.md",
		"summary.txt",
		"working memory",
	}

	for _, keyword := range keywords {
		assert.Contains(t, contentStr, keyword,
			"CLAUDE.md should contain instruction keyword: %s", keyword)
	}
}

func TestEnsureGitignore_PreservesNewlines(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	gitignorePath := filepath.Join(repoPath, ".gitignore")

	// Create .gitignore without trailing newline
	existingContent := "*.log"
	err = os.WriteFile(gitignorePath, []byte(existingContent), 0644)
	require.NoError(t, err)

	err = EnsureGitignore(repoPath)
	require.NoError(t, err)

	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)

	contentStr := string(content)

	// Should have added .claude/ entry
	assert.Contains(t, contentStr, ".claude/")

	// Should have proper formatting (newlines)
	lines := strings.Split(strings.TrimSpace(contentStr), "\n")
	assert.GreaterOrEqual(t, len(lines), 2)
}
