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

Yggdrasil provides an automated installation script that detects your operating system and architecture, downloads the latest pre-compiled binary, and configures your shell environment.

Run the following command in your terminal:

```bash
curl -sSL [https://raw.githubusercontent.com/jordinkolman/yggdrasil/main/install.sh](https://raw.githubusercontent.com/jordinkolman/yggdrasil/main/install.sh) | bash
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
