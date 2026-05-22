#!/bin/bash

# Check if already within tmux session to prevent infinite loop
if [ -n "$TMUX" ]; then
  return 0
fi

echo "Select Session Type:"
echo "1) Terminal Session"
echo "2) Coding Session"
echo "3) Standard Shell (Bypass)"
read -p "Choice (1/2/3): " choice

case $choice in
  1)
    # TERMINAL SESSION
    # Start tmux session with 3 panes, 1 large and 2 vertically stacked
    tmux new-session -d -s "Terminal"
    tmux split-window -h -l 33% -t "Terminal:0.0"
    tmux split-window -v -l 1% -t "Terminal:0.1"
    # Run script to fetch upcoming due tasks from Notion database (notion_daily.sh)
    tmux send-keys -t "Terminal:0.2" "notion_daily.sh &" C-m
    tmux select-pane -t "Terminal:0.0"
    tmux attach-session -t "Terminal"
    ;;
  2)
    # CODING SESSION
    # cd into workspace directory, where all coding projects are stored
    WORKSPACE_DIR="$HOME/workspace"
    cd "$WORKSPACE_DIR" || exit
    # Start tmux session with 4 panes, 1 large and 3 vertically stacked
    tmux new-session -d -s "Coding" -c "$WORKSPACE_DIR"
    tmux split-window -h -l 33% -t "Coding:0.0" -c "$WORKSPACE_DIR"
    tmux split-window -v -l 50% -t "Coding:0.1" -c "$WORKSPACE_DIR"
    tmux split-window -v -l 25% -t "Coding:0.2" -c "$WORKSPACE_DIR"
    # Launch LunarVim in the large left pane
    tmux send-keys -t "Coding:0.0" "lvim ." C-m
    # Run script to fetch upcoming due tasks from Notion database (notion_daily.sh)
    tmux send-keys -t "Coding:0.3" "notion_daily.sh &" C-m
    # Return focus to LunarVim
    tmux select-pane -t "Coding:0.0"
    tmux attach-session -t "Coding"
    ;;
  *)
    # Bypass selection and drop into standard terminal
    echo "Starting standard shell."
    ;;
esac
