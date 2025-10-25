# Shell integration for claude-workspace
# Add this to your ~/.zshrc or ~/.bashrc

# Interactive clone selector with auto-cd
cwc() {
  local selected_path
  selected_path=$(cw clones -i "$@")
  if [ -n "$selected_path" ] && [ -d "$selected_path" ]; then
    cd "$selected_path" || return 1
    echo "📂 Changed to: $selected_path"
  fi
}

# Alias for quick access
alias cwcd='cwc'
