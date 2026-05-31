# Yggdrasil

An interactive, configuration-driven Tmux session manager and workspace navigator. 

Built in Go using the [Charmbracelet Bubble Tea](https://charm.land/bubbles) TUI framework, Yggdrasil replaces heavy bash scripts with a fast, compiled native binary. It dynamically reads user-defined session layouts and orchestrates complex Tmux environments on the fly.

## Architecture: The Shell Hook Pattern

Because Go binaries execute as child processes, they cannot natively alter the state (like the current working directory) of the parent shell. 


Yggdrasil solves this by decoupling the UI from the execution:
1. The Go application renders the TUI strictly to `stderr`.

2. Once a workspace and session layout are selected, the application exits and prints the raw Bash/Tmux commands to `stdout`.
3. A lightweight shell wrapper catches this output and evaluates it in the parent shell.

## Installation

Yggdrasil is distributed as a pre-built binary. Install it via your preferred package manager:

### macOS / Linux (Homebrew)
```bash
brew install jordinkolman/tap/yggdrasil
```

### Debian / Ubuntu (APT)
```bash
# Add the repository key and list (update URLs based on your hosting provider)
curl -fsSL [https://your-repo-url.com/gpg](https://your-repo-url.com/gpg) | sudo tee /usr/share/keyrings/yggdrasil-archive-keyring.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/yggdrasil-archive-keyring.gpg] [https://your-repo-url.com/apt](https://your-repo-url.com/apt) stable main" | sudo tee /etc/apt/sources.list.d/yggdrasil.list

sudo apt-get update
sudo apt-get install yggdrasil
```

### Shell Configuration
Once installed, add the shell hook to the bottom of your `~/.bashrc` to initialize the wrapper and trigger it on startup:

```bash
# Initialize Yggdrasil shell wrapper
eval "$(yggdrasil init bash)"

# Launch Yggdrasil on startup if not already inside a tmux session
if [ -z "$TMUX" ]; then
    ygg
fi
```


## Configuration

On first run, Yggdrasil generates a default configuration file at `~/.config/yggdrasil.yaml`. 
WARNING: Always verify third-party config files before copying. Third-party configurations can be used by malicious actors to gain access to your system.

You can define your global editor and map out unlimited Tmux sessions, windows, and pane splits. Use the `{{editor}}` tag in any command block to dynamically inject your preferred tooling.


```yaml
settings:
  editor: lvim
sessions:
  - name: "[New] Coding Session"
    description: "1 Large Pane (Editor) + 3 Vertically Stacked"
    windows:
      - name: "main"
        panes:
          - split: "vertical"
            size: "33%"
            command: "{{editor}} ."
          - split: "horizontal"
            size: "50%"
            command: ""
          - split: "horizontal"
            size: "25%"
            command: "notion_daily.sh"
```


## Usage

Run the wrapper command in your terminal:
```bash
ygg
```
