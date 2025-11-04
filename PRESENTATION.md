# claudew
### Multi-tasking with Claude Code Made Easy

---

## The Problem

**You can only work on one thing at a time with Claude**

- Close terminal = lose your Claude session
- Switch to different feature = start Claude from scratch
- Work on multiple things = messy clone management
- "Where was I working on that auth feature again?"

**Plus:** Separate clones needed for each feature (branch conflicts)

**Result:** Can't efficiently multi-task with Claude

---

## The Solution: Persistent Sessions

**claudew keeps multiple Claude sessions running simultaneously**

```
feature-oauth      → Claude running in my-app-1/
bug-fix-login      → Claude running in my-app-2/
refactor-auth      → Claude running in my-app-3/
```

**Switch between them instantly**
- ✅ Sessions stay alive even when you close terminal
- ✅ Auto-manages clone pool
- ✅ Tracks what's where
- ✅ One command to jump between contexts

*(Uses tmux behind the scenes, but you don't need to know tmux)*

---

## Demo: The Main Interface

```bash
$ claudew
```

```
──── WORKSPACES ────
feature-oauth [attached] Add OAuth2 authentication (2h ago)
  📂 my-app-2  🟢 Claude running

bug-fix-login [detached] Fix login redirect issue (yesterday)
  📂 my-app-1  🟢 Claude running in background

refactor-auth [idle] Simplify auth flow (3 days ago)
  📂 (no clone assigned)

──── ACTIONS ────
→ Create new workspace
→ CD to workspace clone
→ Browse clones (3 available, 2 in use, 1 free)
```

**Just select and go** - everything is connected for you

---

## Before claudew

**Working on feature-oauth:**
```bash
cd ~/dev/my-app-oauth-clone
claude
# Working with Claude...
```

**Urgent bug comes in:**
```
# Oh no, need to work on something else!
# Ctrl-C kills Claude session
cd ~/dev/my-app-bug-clone
claude
# Start from scratch, lose all context from feature work
```

**Want to go back to feature?**
```
# Ctrl-C again, kills bug session
cd ~/dev/my-app-oauth-clone  # Wait, which directory was it?
claude
# Start from scratch AGAIN
```

---

## After claudew

**Working on feature-oauth:**
```bash
claudew start feature-oauth
# Claude running, making progress...
```

**Urgent bug comes in:**
```bash
claudew create bug-hotfix
# New Claude session in fresh clone, feature-oauth still running!
# Fix the bug...
```

**Back to feature work:**
```bash
claudew start feature-oauth
# Instantly back, Claude still there with all context
```

**Sessions stay alive. No restart penalty.**

---

## Creating a Workspace

```bash
$ claudew create feature-oauth
```

**claudew sets up everything:**

1. ✅ Finds a free clone (or creates new one)
2. ✅ Creates a persistent session
3. ✅ Starts Claude in the right directory
4. ✅ Tracks the association
5. ✅ Connects you to it

**One command → Ready to work**

---

## Switching: Instant Context Changes

**Jump between work without restarting Claude:**

```bash
# Working on feature-oauth
$ claudew start bug-fix-login
# Instantly in bug-fix-login, Claude already warm

$ claudew start refactor-auth
# Jump to different work, others keep running

$ claudew start feature-oauth
# Back to where you were
```

**No waiting, no context loss, no mental overhead**

---

## Real Workflow Example

**Monday morning:**
```bash
$ claudew
> Select: feature-oauth [detached]
# Pick up exactly where you left off Friday
# Claude still has your conversation history
```

**2pm - Urgent bug:**
```bash
claudew create bug-hotfix-login
# Fresh environment in seconds
# Fix bug, ship it
```

**3pm - Back to feature:**
```bash
claudew start feature-oauth
# Right back to feature work, no setup
```

**Sessions survive terminal restarts, sleep, everything**

---

## Clone Management: Automatic

**Before:**
```bash
~/dev/
  my-app-oauth/          # Which workspace?
  my-app-clone-2/        # What's this for?
  my-app-old/            # Still needed?
  my-app-test/           # ???
```

**After:**
```bash
~/dev/my-app-clones/
  my-app-1/    # Managed by claudew
  my-app-2/    # Assigned automatically
  my-app-3/    # Freed when workspace stops

$ claudew clones
my-app-1  [free]
my-app-2  [feature-oauth]
my-app-3  [bug-fix-login]
```

**Pool of numbered clones, automatically assigned/freed**

---

## Session Persistence: The Magic

**Close terminal? Sessions keep running:**
```bash
$ claudew start feature-oauth
# Working with Claude...
# Close terminal, go home

# Next day, different terminal
$ claudew start feature-oauth
# Claude session still there, conversation intact!
```

**Computer sleep? No problem:**
```bash
# Friday afternoon
claudew start feature-oauth
# Close laptop, go home

# Monday morning
claudew start feature-oauth
# Session survived the weekend
```

---

## Bonus: Context Preservation

**When you DO need to restart Claude (context full):**

```bash
$ claudew restart feature-oauth
```

Prompts you to save progress:
```
Current continuation:
Implemented OAuth callback handler. Added tests.

Enter new continuation:
> Completed OAuth callback
> Next: Add token refresh logic
```

**Next session:** Claude sees your notes

**But usually you just detach/reattach - no restart needed!**

---

## Getting Started

**1. Install:**
```bash
go install github.com/pmossman/claudew@latest
```

**2. Initialize:**
```bash
claudew init
claudew install-shell  # Enables cd integration
```

**3. Register your repo:**
```bash
claudew add-remote my-app git@github.com:company/app.git \
  --clone-dir ~/dev/my-app-clones
```

**4. Create your first workspace:**
```bash
claudew create my-first-feature
```

---

## Quick Reference

**Main commands:**
- `claudew` - Interactive menu (use this most)
- `claudew start <name>` - Switch to workspace
- `claudew create <name>` - New workspace
- `claudew stop <name>` - Stop session, free clone

**Navigation:**
- `claudew cd <name>` - Jump to clone directory

**Status:**
- `claudew list` - All workspaces
- `claudew clones` - Clone pool status

**Tip:** `alias cw='claudew'`

---

## Why This Matters

**Before claudew:**
- 😫 One thing at a time
- 💀 Sessions die when terminal closes
- 🐌 Restart Claude constantly
- 🤯 Track clones manually

**After claudew:**
- 🎯 Multi-task effortlessly
- 💪 Sessions persist forever
- ⚡ Instant context switching
- 🧹 Clone management handled

**Focus on coding, not session management**

**Questions?**

GitHub: https://github.com/pmossman/claudew
