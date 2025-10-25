package session

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Manager handles tmux session operations
type Manager struct{}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{}
}

// GetSessionName returns the tmux session name for a workspace
func (m *Manager) GetSessionName(workspaceName string) string {
	return fmt.Sprintf("claude-ws-%s", workspaceName)
}

// Exists checks if a tmux session exists
func (m *Manager) Exists(sessionName string) (bool, error) {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means session doesn't exist
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check tmux session: %w", err)
	}
	return true, nil
}

// Create creates a new tmux session
func (m *Manager) Create(sessionName, repoPath string) error {
	// Create detached session in the repo directory
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", repoPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}
	return nil
}

// Attach attaches to an existing tmux session or creates and attaches if it doesn't exist
func (m *Manager) Attach(sessionName string) error {
	// Check if we're already in a tmux session
	if os.Getenv("TMUX") != "" {
		// We're inside tmux, switch to the session
		cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Not in tmux, attach normally
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SendKeys sends keys to a tmux session
func (m *Manager) SendKeys(sessionName, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, keys, "C-m")
	return cmd.Run()
}

// Kill kills a tmux session
func (m *Manager) Kill(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}

// List returns all tmux sessions
func (m *Manager) List() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// If there are no sessions, tmux returns an error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "no server running") {
				return []string{}, nil
			}
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(sessions) == 1 && sessions[0] == "" {
		return []string{}, nil
	}
	return sessions, nil
}

// CheckTmuxInstalled checks if tmux is installed
func (m *Manager) CheckTmuxInstalled() error {
	cmd := exec.Command("tmux", "-V")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux is not installed. Please install tmux to use claude-workspace")
	}
	return nil
}

// GetSessionState returns the state of a tmux session: "attached", "detached", or "none"
func (m *Manager) GetSessionState(sessionName string) (string, error) {
	// Check if session exists
	exists, err := m.Exists(sessionName)
	if err != nil {
		return "", err
	}
	if !exists {
		return "none", nil
	}

	// Check if session is attached
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}:#{session_attached}", "-f", fmt.Sprintf("#{==:#{session_name},%s}", sessionName))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get session state: %w", err)
	}

	// Parse output: "session-name:N" where N is the number of attached clients
	// N = 0 means detached, N > 0 means attached (can be multiple clients)
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) >= 2 {
		attachedCount, err := strconv.Atoi(parts[1])
		if err == nil && attachedCount > 0 {
			return "attached", nil
		}
		return "detached", nil
	}

	return "none", nil
}

// SetStatusLine customizes the tmux status line for a session
func (m *Manager) SetStatusLine(sessionName, statusLeft, statusRight string) error {
	// Set status line options for this session
	commands := [][]string{
		{"tmux", "set-option", "-t", sessionName, "status-left-length", "80"},
		{"tmux", "set-option", "-t", sessionName, "status-left", statusLeft},
		{"tmux", "set-option", "-t", sessionName, "status-right-length", "60"},
		{"tmux", "set-option", "-t", sessionName, "status-right", statusRight},
		{"tmux", "set-option", "-t", sessionName, "status-style", "bg=colour235,fg=colour136"},
		{"tmux", "set-option", "-t", sessionName, "status-interval", "5"}, // Update every 5 seconds for git branch
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set tmux option: %w", err)
		}
	}

	return nil
}
