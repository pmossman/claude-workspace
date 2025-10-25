# claude-workspace

A CLI tool for managing multiple Claude Code sessions across different repository clones with automatic context preservation.

## The Problem

When working with multiple Claude Code agents across different repository clones:
- Hard to track which session is in which directory
- Context dilutes over long sessions (hours/days), requiring manual restarts
- Have to manually re-explain previous context when starting fresh
- Permission approvals reset each session
- Messy terminal windows when scaling beyond 2-3 parallel sessions

## The Solution

`claude-workspace` provides:

✅ **Workspace Management**: Named workspaces tied to specific repo clones
✅ **Context Preservation**: Automatic maintenance of context files across sessions
✅ **Session Isolation**: tmux-based session management with locking
✅ **Interactive Selection**: fzf-powered fuzzy finder to quickly switch between workspaces
✅ **Continuation Prompts**: Seamlessly resume work from where you left off
✅ **Auto-start**: Automatically launches Claude Code when starting a workspace

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
claude-workspace
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
go install github.com/pmossman/claude-workspace@latest
```

Or build from source:

```bash
git clone https://github.com/pmossman/claude-workspace.git
cd claude-workspace
go install
```

## Quick Start

1. **Initialize**
   ```bash
   claude-workspace init
   ```

2. **Create a workspace**
   ```bash
   claude-workspace create feature-auth ~/dev/my-repo
   ```

   Optionally provide an initial summary:
   ```bash
   claude-workspace create feature-auth ~/dev/my-repo --summary "OAuth implementation"
   ```

3. **Start working** (interactive selector)
   ```bash
   claude-workspace
   ```

   Or directly:
   ```bash
   claude-workspace start feature-auth
   ```

4. **Claude automatically maintains context**
   As you work, Claude will update `context.md`, `decisions.md`, `continuation.md`, and `summary.txt` based on instructions in `.claude/CLAUDE.md`.

5. **Resume later**
   When you restart the workspace, the continuation prompt is displayed and copied to your clipboard.

## Commands

```bash
claude-workspace                    # Interactive selector (default)
claude-workspace init               # Initialize configuration
claude-workspace create <name> <path> [--summary "..."]  # Create workspace
claude-workspace start <name>       # Start/attach to workspace
claude-workspace list               # List all workspaces
claude-workspace info <name>        # Show workspace details
claude-workspace archive <name>     # Archive completed workspace
claude-workspace clone <from> <to> <path>  # Clone workspace context
claude-workspace quick              # Quick session (no workspace)
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
claude-workspace create feature-a ~/dev/my-repo

# Alt clone - feature B
claude-workspace create feature-b ~/alt/my-repo
```

### Cloning Context

When branching work from an existing workspace:

```bash
claude-workspace clone feature-a feature-a-v2 ~/dev/my-repo
```

Copies all context files to the new workspace.

### Quick Sessions

For one-off questions without long-term context:

```bash
claude-workspace quick
```

### Archiving

When work is complete:

```bash
claude-workspace archive feature-auth
```

Moves workspace to `~/.claude-workspaces/archived/` and removes `.claude/CLAUDE.md` from repo.

## tmux Primer

You don't need to know tmux to use this tool, but here are useful commands:

- `Ctrl-b d` - Detach from session (keeps it running)
- `Ctrl-b [` - Scroll mode (use arrows, q to exit)
- `tmux ls` - List all sessions

The tool handles session creation, switching, and attachment automatically.

## Troubleshooting

### "Workspace has an active session"

Another session is using this workspace. Either:
- Attach to that session: `claude-workspace start <name>`
- Kill the other session: `tmux kill-session -t claude-ws-<name>`

### fzf not found

Install fzf: `brew install fzf` (macOS) or see https://github.com/junegunn/fzf

### tmux not found

Install tmux: `brew install tmux` (macOS) or your package manager

### Context files not being maintained

Check that `.claude/CLAUDE.md` exists in your repo. If you created the workspace before this tool was updated, regenerate it:

```bash
claude-workspace create <name> <path>  # Will error if exists
# Or manually copy the CLAUDE.md template from another workspace
```

## Contributing

Issues and PRs welcome at https://github.com/pmossman/claude-workspace

## License

MIT
