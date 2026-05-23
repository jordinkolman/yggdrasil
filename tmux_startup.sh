#!/bin/bash

# Check if already within tmux session to prevent infinite loop
if [ -n "$TMUX" ]; then
  if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    exit 0
  else
    return 0
  fi
fi

# Fetch active tmux sessions
active_sessions=$(tmux list-sessions -F "#{session_name}" 2>/dev/null)

# Build menu options
options="[New] Terminal Session\n[New] Coding Session\n[Bypass] Standard Shell"

if [ -n "$active_sessions" ]; then
  options="$options\n$active_sessions"
fi

choice=$(echo -e "$options" | fzf --prompt="Select Session ❯ " --height=40% --layout=reverse --border --info=hidden)

case "$choice" in
  "[New] Terminal Session")
    # TERMINAL SESSION
    while true; do
      read -p $'\e[1;33mEnter session name (default: Terminal): \e[0m' custom_name
      session_name=${custom_name:-Terminal}

      if tmux has-session -t "$session_name" 2>/dev/null; then
        echo -e "\e[1;31mSession '$session_name' already exists. Choose a different name.\e[0m"
      else
        break
      fi
    done

    # Prompt for target directory
    read -e -p $'\e[1;33m Enter target directory (default: ~): \e[0m]' target_input
    TARGET_DIR="${target_input:-$HOME}"
    TARGET_DIR="${TARGET_DIR/#\~/$HOME}"

    mkdir -p "$TARGET_DIR"
    cd "$TARGET_DIR" || exit

    # Start tmux session with 3 panes, 1 large and 2 vertically stacked
    tmux new-session -d -s "$session_name" -c "$TARGET_DIR"
    tmux split-window -h -l 33% -t "$session_name:0.0" -c "$TARGET_DIR"
    tmux split-window -v -l 10% -t "$session_name:0.1" -c "$TARGET_DIR"
    # Run script to fetch upcoming due tasks from Notion database (notion_daily.sh)
    tmux send-keys -t "$session_name:0.2" "notion_daily.sh &" C-m
    tmux select-pane -t "$session_name:0.0"
    tmux attach-session -t "$session_name"
    ;;
  "[New] Coding Session")
    # CODING SESSION
    # cd into workspace directory, where all coding projects are stored
    while true; do
      read -p $'\e[1;33mEnter session name (default: Coding): \e[0m' custom_name
      session_name=${custom_name:-Coding}

      if tmux has-session -t "$session_name" 2>/dev/null; then
        echo -e "\e[1;31mSession '$session_name' already exists. Choose a different name.\e[0m"
      else
        break
      fi
    done

    WORKSPACE_DIR="$HOME/workspace"
    cd "$WORKSPACE_DIR" || exit

    # Directory selection via fzf
    existing_dirs=$(find "$WORKSPACE_DIR" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; 2>/dev/null)
    dir_options="[Create New Project]"
    if [ -n "$existing_dirs" ]; then
      dir_options="$dir_options\n$existing_dirs"
    fi

    dir_choice=$(echo -e "$dir_options" | fzf --prompt="Select Project ❯ " --height=40% --layout=reverse --border --info=hidden)

    if [ "$dir_choice" == "[Create New Project]" ]; then
      read -p $'\e[1;33mEnter new project name: \e[0m' new_project_name
      TARGET_DIR="$WORKSPACE_DIR/${new_project_name:-untitled_project}"
      mkdir -p "$TARGET_DIR"
    elif [ -z "$dir_choice" ]; then
      TARGET_DIR="$WORKSPACE_DIR"
    else
      TARGET_DIR="$WORKSPACE_DIR/$dir_choice"
    fi

    cd "$TARGET_DIR" || exit

    # Start tmux session with 4 panes, 1 large and 3 vertically stacked
    tmux new-session -d -s "$session_name" -c "$TARGET_DIR"
    tmux split-window -h -l 33% -t "$session_name:0.0" -c "$TARGET_DIR"
    tmux split-window -v -l 50% -t "$session_name:0.1" -c "$TARGET_DIR"
    tmux split-window -v -l 25% -t "$session_name:0.2" -c "$TARGET_DIR"
    # Launch LunarVim in the large left pane
    tmux send-keys -t "$session_name:0.0" "lvim ." C-m
    # Run script to fetch upcoming due tasks from Notion database (notion_daily.sh)
    tmux send-keys -t "$session_name:0.3" "notion_daily.sh &" C-m
    # Return focus to LunarVim
    tmux select-pane -t "$session_name:0.0"
    tmux attach-session -t "$session_name"
    ;;
  "[Bypass] Standard Shell" | "")
    # Bypass selection and drop into standard terminal
    echo "Starting standard shell."
    ;;
  *)
    tmux attach-session -t "$choice"
    ;;
esac
