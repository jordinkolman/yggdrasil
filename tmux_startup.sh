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

choice=$(echo -e "$options" | fzf \
  --prompt="Select Session ❯ " \
  --height=50% \
  --layout=reverse \
  --border \
  --info=hidden \
  --preview-window=right:50% \
  --preview '
    case {} in
      "[New] Terminal Session")
        echo -e "\n  \e[1;36mTERMINAL SESSION LAYOUT\e[0m\n"
        echo -e "  [ Large Left Pane ]  [ Small Right Top    ]"
        echo -e "  [                 ]  [ Standard Shell     ]"
        echo -e "  [ Interactive     ]  ----------------------"
        echo -e "  [ Shell           ]  [ Small Right Bottom ]"
        echo -e "  [                 ]  [ notion_daily.sh    ]"
        ;;
      "[New] Coding Session")
        echo -e "\n  \e[1;35mCODING SESSION LAYOUT\e[0m\n"
        echo -e "  [ Large Left Pane ]  [ Small Right Top    ]"
        echo -e "  [                 ]  [ Standard Shell     ]"
        echo -e "  [ LunarVim        ]  ----------------------"
        echo -e "  [ (lvim .)        ]  [ Small Right Middle ]"
        echo -e "  [                 ]  [ Standard Shell     ]"
        echo -e "  [                 ]  ----------------------"
        echo -e "  [                 ]  [ Small Right Bottom ]"
        echo -e "  [                 ]  [ notion_daily.sh    ]"
        ;;
      "[Bypass] Standard Shell")
        echo -e "\n  \e[1;32mBYPASS INITIALIZATION\e[0m\n"
        echo -e "  Drops directly into a standard, un-multiplexed"
        echo -e "  interactive bash shell."
        ;;
      "")
        echo "No selection."
        ;;
      *)
        echo -e "\n  \e[1;33mEXISTING TMUX SESSION: {}\e[0m\n"
        echo -e "  Active Windows & Panes:\n"
        tmux list-windows -t "{}" 2>/dev/null || echo "  Could not fetch session details."
        ;;
    esac
  ')

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
    read -e -p $'\e[1;33m Enter target directory (default: ~): \e[0m' target_input
    TARGET_DIR="${target_input:-$HOME}"
    TARGET_DIR="${TARGET_DIR/#\~/$HOME}"

    mkdir -p "$TARGET_DIR"
    cd "$TARGET_DIR" || exit

    TARGET_DIR="$PWD"

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
    export WORKSPACE_DIR="$HOME/workspace"
    mkdir -p "$WORKSPACE_DIR"
    
    while true; do
      read -p $'\e[1;33mEnter session name (default: Coding): \e[0m' custom_name
      session_name=${custom_name:-Coding}

      if tmux has-session -t "$session_name" 2>/dev/null; then
        echo -e "\e[1;31mSession '$session_name' already exists. Choose a different name.\e[0m"
      else
        break
      fi
    done

    CURRENT_BROWSE_DIR="$WORKSPACE_DIR"

    # Directory selection via fzf
    while true; do
      existing_dirs=$(find "$CURRENT_BROWSE_DIR" -mindepth 1 -maxdepth 1 -type d \( -name '.*' -o -name 'node_modules' -o -name 'venv' \) -prune -o -type d -exec basename {} \; | sort 2>/dev/null)
  
      dir_options="[Select This Directory]\n[Create New Project Here]"

      if [ "$CURRENT_BROWSE_DIR" != "$WORKSPACE_DIR" ]; then
        dir_options="$dir_options\n[Go Up One Level]"
      fi

      if [ -n "$existing_dirs" ]; then
        dir_options="$dir_options\n$existing_dirs"
      fi

      display_path="${CURRENT_BROWSE_DIR/#$HOME/\~}"

      dir_choice=$(echo -e "$dir_options" | fzf \
        --prompt="Browse ❯ $display_path/ ❯ " \
        --height=50% \
        --layout=reverse \
        --border \
        --info=hidden \
        --preview-window=right:50% \
        --preview '
          dir={}
          if [ "$dir" = "[Select This Directory]" ]; then
            echo -e "\n \e[1;32mUse Current Directory\e[0m\n\n  Target: '"$CURRENT_BROWSE_DIR"'"
          elif [ "$dir" = "[Create New Project Here]" ]; then
            echo -e "\n  \e[1;33mCreate New Project Directory\e[0m\n\n  Creates a new subdirectory inside:\n  '"$CURRENT_BROWSE_DIR"' "
          elif [ "$dir" = "[Go Up One Level]" ]; then
            echo -e "\n \e[1;36mGo Up\e[0m\n\n  Return to: '"$(dirname "$CURRENT_BROWSE_DIR")"'"
          elif [ -n "$dir" ]; then
           echo -e "\n  \e[1;34mContents of $dir:\e[0m\n"
            ls -la --color=always '"$CURRENT_BROWSE_DIR"'/"$dir" 2>/dev/null || ls -la '"$CURRENT_BROWSE_DIR"'/"$dir"
          fi
        ')

      case "$dir_choice" in 
        "[Select This Directory]")
          TARGET_DIR="$CURRENT_BROWSE_DIR"
          break
          ;;
        "[Create New Project Here]")
          read -p $'\e[1;33mEnter new project name: \e[0m' new_project_name
          TARGET_DIR="$CURRENT_BROWSE_DIR/${new_project_name:-untitled_project}"
          mkdir -p "$TARGET_DIR"
          break
          ;;
        "[Go Up One Level]")
          CURRENT_BROWSE_DIR=$(dirname "$CURRENT_BROWSE_DIR")
          ;;
        "")
          TARGET_DIR="$WORKSPACE_DIR"
          break
          ;;
        *)
          CURRENT_BROWSE_DIR="$CURRENT_BROWSE_DIR/$dir_choice"
          ;;
      esac
    done

    cd "$TARGET_DIR" || exit
    TARGET_DIR="$PWD"

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
