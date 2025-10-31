# claudew

**Manage multiple Claude Code sessions across different repos with automatic context preservation**

Stop losing track of which Claude is working on which repository. Keep your AI agents organized, persistent, and context-aware across long-running projects.

[![Tests](https://img.shields.io/badge/tests-130%20passing-brightgreen)]() [![Coverage](https://img.shields.io/badge/coverage-85%25-green)]() [![Security](https://img.shields.io/badge/security-hardened-blue)]()

---

## 🎯 What is it?

A tmux-powered workspace manager for Claude Code that lets you:
- **Run multiple Claude sessions** across different git repos simultaneously
- **Preserve context automatically** - Claude maintains progress notes, decisions, and handoff prompts
- **Switch instantly** with an interactive fuzzy finder (fzf)
- **Resume seamlessly** - pick up exactly where you left off, even days later

Think of it as "tmux + Claude Code + smart context management" in one tool.

---

## 🎬 Quick Demo

```bash
# Create a workspace tied to a repository
$ claudew create feature-auth ~/code/api-server --summary "Implementing OAuth2"
✓ Created workspace 'feature-auth'
✓ Generated .claude/CLAUDE.md with context instructions
✓ Added .claude/ to .gitignore

# Interactive selector shows all workspaces with status
$ cw

  ──── WORKSPACES ────
  feature-auth [detached] Implementing OAuth2 (5 min ago)
  bug-db-leak [attached] Fixing connection pool leak (2 hours ago)
  refactor-api [idle] Simplifying REST endpoints (yesterday)

  ──── ACTIONS ────
  → Create new workspace
  → Archive workspace
  → Browse clones (3 available)

# Select a workspace, and you're dropped into a tmux session:
$ claudew start feature-auth

═══════════════════════════════════════════
  Workspace: feature-auth
  Repository: ~/code/api-server
  Summary: Implementing OAuth2
═══════════════════════════════════════════

📋 CONTINUATION PROMPT:
─────────────────────────────────────────
Continue OAuth2 implementation. We finished:
- JWT token generation
- Refresh token rotation

Next steps:
- Implement token revocation endpoint
- Add rate limiting to auth endpoints
─────────────────────────────────────────

✓ Copied to clipboard

[feature-auth] api-server @ feature-branch | Implementing OAuth2

# Claude Code automatically starts...
# Work for hours, then detach with Ctrl-b d
# Context is automatically saved to continuation.md

# Later, switch to another workspace
$ cw
# Select bug-db-leak...
# Instantly dropped into that session with full context
```

---

## 💡 The Problem

When working with multiple Claude Code agents across different repositories:

- ❌ Hard to track which Claude is working in which directory
- ❌ Context dilutes over long sessions, requiring manual context re-entry
- ❌ Permission approvals reset every session
- ❌ Messy terminal windows when juggling 3+ parallel sessions
- ❌ No easy way to resume after Claude crashes or you need to restart

## ✨ The Solution

**claudew** provides:

- ✅ **Named workspaces** tied to specific repository directories
- ✅ **Automatic context preservation** - Claude maintains handoff prompts
- ✅ **tmux-based sessions** that survive terminal closures
- ✅ **Interactive selection** with fuzzy search (fzf)
- ✅ **Session locking** to prevent workspace collisions
- ✅ **Git clone management** for parallel work on the same repo
- ✅ **Status bar** showing workspace name, repo path, branch, and summary

---

## 🎯 Perfect For

**Parallel feature development:**
```bash
# Clone 1: Main feature
claudew create oauth-server ~/dev/api-clone-1

# Clone 2: Dependent microservice
claudew create oauth-client ~/dev/api-clone-2

# Clone 3: Testing & docs
claudew create oauth-testing ~/dev/api-clone-3
```

**Long-running refactors:**
```bash
# Week 1: Start major refactor
claudew create db-migration ~/code/app --summary "Postgres to MongoDB migration"
# ... work with Claude for hours ...
# Ctrl-b d to detach

# Week 2: Resume with full context
claudew start db-migration
# Claude reads continuation.md and knows exactly where you left off
```

**Rapid context switching:**
```bash
# Working on feature A
claudew start feature-auth

# Urgent bug reported!
# Ctrl-b d to detach
claudew start bug-production-leak
# ... fix bug with dedicated Claude ...

# Back to feature work
claudew start feature-auth
# Right back where you left off
```

**Team collaboration:**
```bash
# Fork a teammate's workspace when taking over
claudew fork johns-feature my-continuation ~/code/repo

# All context (decisions, research, progress) is preserved
```

---

## Features

### Context Files (Automatically Maintained by Claude)

Each workspace maintains:
- `context.md` - Current progress, objectives, and blockers
- `decisions.md` - User corrections and clarifications (critical memory)
- `continuation.md` - Handoff prompt for next session
- `summary.txt` - One-line workspace description
- `research/` - Code exploration findings

Claude automatically maintains these files based on instructions in the workspace's `CLAUDE.md`.

### Session Management

- **tmux integration**: Each workspace runs in its own tmux session
- **Session locking**: Prevents multiple agents from clobbering the same workspace
- **Auto-start**: Claude Code launches automatically when you start a workspace
- **Continuation prompts**: Displayed prominently (and copied to clipboard) when starting

### Interactive Selector

```bash
# Just run without arguments
claudew
```

Opens an fzf selector showing:
- Workspace names and summaries
- Status (active/idle/archived)
- Last active time
- Preview pane with continuation prompt and context

## Installation

### Prerequisites

- Go 1.21+ (for building from source)
- tmux: `brew install tmux` (macOS) or your package manager
- fzf: `brew install fzf` (for interactive selector)

### Install

```bash
go install github.com/pmossman/claudew@latest
```

Or build from source:

```bash
git clone https://github.com/pmossman/claudew.git
cd claudew
go install
```

## Quick Start

1. **Initialize and install shell integration**
   ```bash
   claudew init
   claudew install-shell
   ```

   Activate the shell integration:
   ```bash
   source ~/.zshrc  # or ~/.bashrc for bash
   ```

2. **Create a workspace**
   ```bash
   # Using claudew (create 'alias cw=claudew' for shorter typing)
   claudew create feature-auth ~/dev/my-repo --summary "OAuth implementation"
   ```

3. **Start working** (interactive selector)
   ```bash
   claudew    # Just type 'cw' - that's it!
   ```

   Use arrow keys to select a workspace, press Enter. Or directly:
   ```bash
   claudew start feature-auth
   ```

4. **Claude automatically maintains context**
   As you work, Claude updates `context.md`, `decisions.md`, `continuation.md`, and `summary.txt` based on instructions in `.claude/CLAUDE.md`.

5. **Detach and resume later**
   ```bash
   # Press Ctrl-b d to detach (keeps Claude running)
   # Days later...
   claudew start feature-auth
   # Continuation prompt appears with full context!
   ```

## Commands

```bash
claudew                                  # Interactive selector (default)
claudew init                             # Initialize configuration
claudew create <name> <path> [--summary "..."]  # Create workspace
claudew start <name>                     # Start/attach to workspace
claudew list                             # List all workspaces
claudew info <name>                      # Show workspace details
claudew archive <name>                   # Archive completed workspace
claudew fork <from> <to> <path>          # Fork workspace context to new workspace
claudew install-shell                    # Install shell integration and tab completion

# Full command is also available
claudew <command>
```

## How It Works

### Directory Structure

```
~/.claude-workspaces/
├── config.json                    # Workspace registry
├── feature-auth/                  # Example workspace
│   ├── context.md                 # Current progress
│   ├── decisions.md               # User corrections
│   ├── continuation.md            # Next session prompt
│   ├── summary.txt                # One-line description
│   └── research/                  # Code exploration notes
└── bug-fix-123/
    └── ...
```

### Workspace Setup

When you create a workspace:
1. Creates workspace directory with context files
2. Generates `.claude/CLAUDE.md` in your repo with instructions for Claude
3. Adds `.claude/` to `.gitignore` (won't be committed)
4. Registers workspace in config

### Starting a Session

When you start a workspace:
1. Creates/attaches to tmux session named `claude-ws-<name>`
2. Changes to the repository directory
3. Displays continuation prompt (copies to clipboard)
4. Creates lock file (if locking enabled)
5. Auto-starts Claude Code

### Claude's Behavior

The generated `.claude/CLAUDE.md` instructs Claude to:
- **On startup**: Read continuation.md and decisions.md
- **During work**: Update context.md after each task
- **When corrected**: Immediately record to decisions.md
- **After research**: Write findings to research/<topic>.md
- **Periodically**: Update continuation.md (every 30min)
- **As learned**: Update summary.txt with better description

## Configuration

Config is stored at `~/.claude-workspaces/config.json`:

```json
{
  "workspaces": {
    "feature-auth": {
      "name": "feature-auth",
      "repo_path": "/Users/you/dev/repo",
      "created_at": "2025-10-24T10:30:00Z",
      "last_active": "2025-10-24T14:22:00Z",
      "status": "idle",
      "session_pid": 0
    }
  },
  "settings": {
    "workspace_dir": "/Users/you/.claude-workspaces",
    "auto_start_claude": true,
    "require_session_lock": true,
    "claude_command": "claude"
  }
}
```

## Tips

### Multiple Clones of Same Repo

Perfect for working on independent features in parallel:

```bash
# Dev clone - feature A
claudew create feature-a ~/dev/my-repo

# Alt clone - feature B
claudew create feature-b ~/alt/my-repo
```

### Forking Workspaces

When branching work from an existing workspace:

```bash
claudew fork feature-a feature-a-v2 ~/dev/my-repo
```

Copies all context files from the source workspace to the new workspace.

### Archiving

When work is complete:

```bash
claudew archive feature-auth
```

Moves workspace to `~/.claude-workspaces/archived/` and removes `.claude/CLAUDE.md` from repo.

## tmux Configuration for Beginners

This tool uses tmux to manage persistent sessions. While `claudew` handles most tmux complexity for you, configuring tmux will greatly improve your experience.

### Recommended tmux Configuration

Create or edit `~/.tmux.conf` with these settings:

```bash
# Enable mouse mode - allows scrolling with mouse wheel and clicking to select panes
set -g mouse on

# Increase scrollback buffer size (default is 2000 lines)
set -g history-limit 10000

# Use vi mode for copy mode (optional, but helpful if you like vi keys)
setw -g mode-keys vi

# Don't automatically rename windows based on running command
set-option -g allow-rename off

# Start window numbering at 1 instead of 0 (easier to reach on keyboard)
set -g base-index 1
setw -g pane-base-index 1

# Reduce escape time (helps with vim/neovim responsiveness)
set -sg escape-time 10

# Enable 256 color support
set -g default-terminal "screen-256color"

# Easier pane splitting with | and -
bind | split-window -h
bind - split-window -v
unbind '"'
unbind %

# Easy config reload
bind r source-file ~/.tmux.conf \; display-message "Config reloaded!"
```

After creating this file, reload it in any active tmux session:
```bash
# Inside a tmux session, press Ctrl-b then type:
:source-file ~/.tmux.conf

# Or use the shortcut (if you added the config above):
# Press Ctrl-b then r
```

### Essential tmux Commands

**Session Management:**
- `Ctrl-b d` - Detach from session (keeps it running in background)
- `tmux ls` - List all sessions (from outside tmux)
- `tmux attach -t <session-name>` - Reattach to a session

**Scrolling & Copy Mode:**
- **Mouse wheel** - Scroll up/down (if mouse mode enabled)
- `Ctrl-b [` - Enter copy mode manually (use arrows to scroll, `q` to exit)
- `Ctrl-b PgUp` - Enter copy mode and scroll up one page

**Pane Management (if you split your screen):**
- `Ctrl-b |` - Split vertically (with recommended config)
- `Ctrl-b -` - Split horizontally (with recommended config)
- `Ctrl-b arrow` - Move between panes
- `Ctrl-b x` - Close current pane

**Window Management:**
- `Ctrl-b c` - Create new window
- `Ctrl-b n` - Next window
- `Ctrl-b p` - Previous window
- `Ctrl-b ,` - Rename current window

### Why Mouse Mode is Essential

Without mouse mode enabled, scrolling in tmux is painful (`Ctrl-b [`, arrow keys, then `q` to exit). With mouse mode:
- **Just scroll** with your mouse wheel - it works naturally
- **Click** to select panes (if you use split screens)
- **Click and drag** to select text
- **Double-click** to select a word

### Quick Setup

```bash
# Create the recommended config
cat > ~/.tmux.conf << 'EOF'
set -g mouse on
set -g history-limit 10000
setw -g mode-keys vi
set-option -g allow-rename off
set -g base-index 1
setw -g pane-base-index 1
set -sg escape-time 10
set -g default-terminal "screen-256color"
bind | split-window -h
bind - split-window -v
unbind '"'
unbind %
bind r source-file ~/.tmux.conf \; display-message "Config reloaded!"
EOF

# Reload all existing claudew sessions
tmux source-file ~/.tmux.conf
```

The tool handles session creation, switching, and attachment automatically - you just need to configure tmux once for a better experience!

## Troubleshooting

### "Workspace has an active session"

Another session is using this workspace. Either:
- Attach to that session: `claudew start <name>`
- Kill the other session: `tmux kill-session -t claude-ws-<name>`

### fzf not found

Install fzf: `brew install fzf` (macOS) or see https://github.com/junegunn/fzf

### tmux not found

Install tmux: `brew install tmux` (macOS) or your package manager

### Context files not being maintained

Check that `.claude/CLAUDE.md` exists in your repo. If you created the workspace before this tool was updated, regenerate it:

```bash
claudew create <name> <path>  # Will error if exists
# Or manually copy the CLAUDE.md template from another workspace
```

### Mouse scrolling doesn't work in tmux

Enable mouse mode in your `~/.tmux.conf`:
```bash
set -g mouse on
```

Then reload the config inside tmux: `Ctrl-b` then `:source-file ~/.tmux.conf`

See the [tmux Configuration](#tmux-configuration-for-beginners) section above for the full recommended setup.

### Can't scroll back far enough in tmux

Increase the scrollback buffer in `~/.tmux.conf`:
```bash
set -g history-limit 10000  # Default is usually 2000
```

## Code Quality & Security

This tool is built with production-grade practices:

**🧪 Comprehensive Testing**
- 130+ unit tests across all packages
- 85% code coverage
- Integration tests for git and tmux operations
- Test coverage for edge cases and error paths

**🔒 Security Hardening**
- Input validation prevents path traversal attacks
- Shell argument escaping prevents command injection
- Workspace name validation blocks problematic characters
- Defense-in-depth approach with multiple validation layers

**📦 Clean Architecture**
- Separated concerns (config, workspace, git, session, template packages)
- Minimal dependencies (just Cobra for CLI, testify for tests)
- Well-documented code with clear interfaces
- Refactored for maintainability (small, focused functions)

**🛡️ Validated Inputs**
- Workspace names validated (no spaces, special chars, or path separators)
- All file paths checked for traversal attempts
- Repository paths escaped in shell commands
- Clone and remote URLs sanitized

Run the tests yourself:
```bash
go test -v ./...
```

## Contributing

Issues and PRs welcome at https://github.com/pmossman/claudew

## License

MIT
