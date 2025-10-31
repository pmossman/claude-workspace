#!/bin/sh
# claudew shell integration
# This file is installed to ~/.claudew/shell-integration.sh by 'claudew install-shell'

claudew() {
  # Pass through completion requests directly without capturing output
  if [ "$1" = "__complete" ]; then
    command claudew "$@"
    return $?
  fi

  # Only capture output for commands that may use CD: marker
  # All other commands pass through directly for real-time output
  case "$1" in
    cd|clones|select|"")
      # These commands might output CD: marker for navigation
      local output
      output=$(command claudew "$@" 2>&1)
      local exit_code=$?

      # Check if output contains CD: marker (for clone navigation)
      # Use CD::: as delimiter to handle paths with colons
      if echo "$output" | grep -q "^CD:::"; then
        local clone_path=$(echo "$output" | grep "^CD:::" | sed 's/^CD::://')
        if [ -n "$clone_path" ]; then
          if [ -d "$clone_path" ]; then
            cd "$clone_path" || {
              echo "❌ Error: Failed to change directory to: $clone_path" >&2
              return 1
            }
            echo "📂 Changed to: $clone_path"
            return 0
          else
            echo "❌ Error: Directory does not exist: $clone_path" >&2
            return 1
          fi
        fi
      fi

      # Otherwise, just display the output normally
      echo "$output"
      return $exit_code
      ;;
    *)
      # All other commands: pass through directly (no output buffering)
      command claudew "$@"
      return $?
      ;;
  esac
}

# Short alias for convenience
alias cw='claudew'
