package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type ClaudeMdData struct {
	WorkspaceName string
	WorkspaceDir  string
	RepoPath      string
}

const claudeMdTemplate = `# Workspace: {{.WorkspaceName}}
# Workspace Directory: {{.WorkspaceDir}}
# Repository: {{.RepoPath}}

## 🚨 CRITICAL: Context Management Protocol

You are in a managed workspace. **You MUST maintain context files** to preserve your work across sessions.

### Required Files (Update These Proactively)

**1. {{.WorkspaceDir}}/context.md** - Your working memory
- Update after completing any significant task or subtask
- Include: current objective, what's done, what's next, current blockers
- Keep under 500 words, rewrite if it grows too long
- Update frequency: Every 20-30 minutes or after major milestones

**2. {{.WorkspaceDir}}/decisions.md** - User corrections & requirements
- **IMMEDIATELY** write here when user corrects you or clarifies requirements
- Format: ` + "`## [Timestamp] Topic\\nUser clarified: <exact correction>\\nReason: <why this matters>\\n`" + `
- This is your most important memory - never lose user corrections

**3. {{.WorkspaceDir}}/research/<topic>.md** - Code exploration findings
- BEFORE exploring code, check if research/<topic>.md exists
- AFTER researching unfamiliar systems, write comprehensive notes
- One file per major topic (e.g., "auth-flow.md", "database-migrations.md")
- Include: key files, important patterns, gotchas discovered

**4. {{.WorkspaceDir}}/continuation.md** - Handoff to next session
- Update every 30 minutes AND before you expect the session might end
- Format: "Working on: [X]. Completed: [Y]. Next: [specific actionable steps]"
- Be specific enough that a fresh session can continue seamlessly

**5. {{.WorkspaceDir}}/summary.txt** - One-line workspace description
- Keep a one-line summary of what this workspace is for
- Update as your understanding of the work evolves
- Max 60 characters, descriptive but concise
- Format: "Brief description of the feature/bug/work"

### Session Startup Protocol

**IMMEDIATELY at session start:**
1. Read continuation.md to understand current work
2. Read decisions.md to recall user corrections
3. Acknowledge what you're working on
4. Check context.md for additional details if needed

### During Work

- After each TODO item completion → Update context.md
- User corrects you → STOP and update decisions.md first
- About to research code → Check research/ for existing notes
- Every 30 min or major milestone → Update continuation.md
- Discovering architectural patterns → Write to research/
- As you gain context → Update summary.txt

### Context Management & Restarts

**Monitor your context usage throughout the session:**
- If your context window reaches 70-80% full, proactively suggest a restart to the user
- Before accepting a restart, ALWAYS update continuation.md with current progress
- Format: "Context is at X%. Consider restarting with: cw restart {{.WorkspaceName}}"
- A restart preserves all workspace state while giving you a fresh context window

**Why restart matters:**
- Prevents context compaction from dropping important instructions
- Ensures CLAUDE.md instructions remain fully in context
- Allows you to reload with a focused continuation prompt
- Maintains workspace continuity without losing progress

**When to suggest restart:**
- Context >70%: Mention it's getting full, offer to continue or restart
- Context >85%: Strongly recommend restart before continuing
- Before long tasks: If context is >50% and starting something complex

### These files are FOR YOU, not the user
Don't ask permission to maintain them. Do it proactively.
The user won't read these - they're your memory system.
`

// GenerateClaudeMd generates a CLAUDE.md file in the repo's .claude directory
func GenerateClaudeMd(workspaceName, workspaceDir, repoPath string) error {
	claudeDir := filepath.Join(repoPath, ".claude")
	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")

	// Create .claude directory if it doesn't exist
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("claude_md").Parse(claudeMdTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := ClaudeMdData{
		WorkspaceName: workspaceName,
		WorkspaceDir:  workspaceDir,
		RepoPath:      repoPath,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write CLAUDE.md file
	if err := os.WriteFile(claudeMdPath, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	return nil
}

// EnsureGitignore ensures .claude/ is in the repo's .gitignore
func EnsureGitignore(repoPath string) error {
	gitignorePath := filepath.Join(repoPath, ".gitignore")

	// Read existing gitignore if it exists
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	gitignoreStr := string(content)

	// Check if .claude/ is already in gitignore
	if strings.Contains(gitignoreStr, ".claude/") {
		return nil
	}

	// Append .claude/ to gitignore
	if len(gitignoreStr) > 0 && !strings.HasSuffix(gitignoreStr, "\n") {
		gitignoreStr += "\n"
	}
	gitignoreStr += "\n# Claude workspace files\n.claude/\n"

	if err := os.WriteFile(gitignorePath, []byte(gitignoreStr), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

// RemoveClaudeMd removes the CLAUDE.md file from the repo
func RemoveClaudeMd(repoPath string) error {
	claudeMdPath := filepath.Join(repoPath, ".claude", "CLAUDE.md")
	err := os.Remove(claudeMdPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove CLAUDE.md: %w", err)
	}
	return nil
}
