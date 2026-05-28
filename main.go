package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"go.yaml.in/yaml/v3"
)

// Read values from yaml config
type Config struct {
	Settings SettingsConfig  `yaml:"settings"`
	Sessions []SessionConfig `yaml:"sessions"`
}

type SettingsConfig struct {
	Editor string `yaml:"editor"`
}

type SessionConfig struct {
	Name    string         `yaml:"name"`
	Desc    string         `yaml:"description"`
	Windows []WindowConfig `yaml:"windows"`
}

type WindowConfig struct {
	Name  string       `yaml:"name"`
	Panes []PaneConfig `yaml:"panes"`
}

type PaneConfig struct {
	Split   string `yaml:"split"`   // e.g., "horizontal" or "vertical"
	Size    string `yaml:"size"`    // e.g., "50%" or "10"
	Command string `yaml:"command"` // The custom script or command to execute
}

type genericItem struct {
	title string
	desc  string
}

// tracks which screen is currently active
type viewState int

const (
	viewSessionSelect viewState = iota // The initial session select menu (Terminal, Coding, Bypass, etc.)
	viewDirBrowse                      // Interactive file explorer
	viewNameInput                      // Text prompt for a new session/project name
)

type model struct {
	// --- UI Components ---
	// Each list will maintain its own cursor positions and filtering states
	sessionList list.Model
	dirList     list.Model
	textInput   textinput.Model

	// --- App State ---
	activeView       viewState
	workspaceDir     string // The base workspace path - ~/workspace by default
	currentBrowseDir string // The directory currently being viewed
	selectedLayout   SessionConfig

	// --- Final Output State ---
	// Final values that will be printed to stdout for bash wrapper to evaluate
	editor      string
	targetDir   string
	sessionName string

	// --- Terminal Dimensions ---
	// window resize events passed from BubbleTea
	width  int
	height int
}

type dirScanMsg []list.Item
type errMsg struct{ err error }

func (s SessionConfig) Title() string       { return s.Name }
func (s SessionConfig) Description() string { return s.Desc }
func (s SessionConfig) FilterValue() string { return s.Name } // The text used for fuzzy filtering

func (i genericItem) Title() string       { return i.title }
func (i genericItem) Description() string { return i.desc }
func (i genericItem) FilterValue() string { return i.title }

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func createDefaultConfig(path string) (Config, error) {
	defaultConfig := Config{
		Settings: SettingsConfig{
			Editor: "vim",
		},
		Sessions: []SessionConfig{
			{
				Name: "[New] Terminal Session",
				Desc: "1 Large Pane + 2 Vertically Stacked",
				Windows: []WindowConfig{
					{
						Name: "main",
						Panes: []PaneConfig{
							{Split: "vertical", Size: "33%", Command: ""}, // Empty command drops to default shell
							{Split: "horizontal", Size: "50%", Command: ""},
							{Split: "horizontal", Size: "100%", Command: ""},
						},
					},
				},
			},
			{
				Name: "[New] Coding Session",
				Desc: "1 Large Pane (Editor) + 3 Vertically Stacked",
				Windows: []WindowConfig{
					{
						Name: "main",
						Panes: []PaneConfig{
							{Split: "vertical", Size: "33%", Command: "{{editor}} ."},
							{Split: "horizontal", Size: "50%", Command: ""},
							{Split: "horizontal", Size: "25%", Command: ""},
						},
					},
				},
			},
		},
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return defaultConfig, fmt.Errorf("failed to marshal default config: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return defaultConfig, fmt.Errorf("failed to write default config: %v", err)
	}

	return defaultConfig, nil
}

// Locates, reads, and parses the yggdrasil.yaml file (~/.config/yggdrasil.yaml)
// If it does not exist, it creates a default one
func loadConfig() (Config, error) {
	var cfg Config

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, fmt.Errorf("could not find home directory: %v", err)
	}

	configDir := filepath.Join(home, ".config")
	configPath := filepath.Join(configDir, "yggdrasil.yaml")

	// Check if file exists, else create the default file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return cfg, fmt.Errorf("failed to create config directory: %v", err)
		}

		return createDefaultConfig(configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse yaml: %v", err)
	}

	return cfg, nil
}

func readDir(targetPath string, isRoot bool) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(targetPath)
		if err != nil {
			return errMsg{err}
		}

		var items []list.Item

		items = append(items, genericItem{
			title: "[Select This Directory]",
			desc:  "Target: " + targetPath,
		})

		items = append(items, genericItem{
			title: "[Create New Project Here]",
			desc:  "Creates a new subdirectory inside " + targetPath,
		})

		if !isRoot {
			items = append(items, genericItem{
				title: "[Go Up One Level]",
				desc:  "Return to: " + filepath.Dir(targetPath),
			})
		}

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}

			name := e.Name()

			if name[0] == '.' || name == "node_modules" || name == "venv" {
				continue
			}

			items = append(items, genericItem{
				title: name,
				desc:  filepath.Join(targetPath, name),
			})
		}

		return dirScanMsg(items)
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// Handle terminal resizing
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sessionList.SetSize(msg.Width, msg.Height)
		m.dirList.SetSize(msg.Width, msg.Height)
		return m, nil

	case dirScanMsg:
		cmd = m.dirList.SetItems(msg)
		return m, cmd

	case errMsg:
		// TODO: Route to a status bar
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {

		// Global Quit
		case "ctrl+c":
			return m, tea.Quit
		// Contextual Quit (q should only quit if not typing a project or session name)
		case "q":
			if m.activeView != viewNameInput {
				return m, tea.Quit
			}
		case "enter":
			switch m.activeView {

			// Step 1: User selects a session type
			case viewSessionSelect:
				if i, ok := m.sessionList.SelectedItem().(SessionConfig); ok {
					m.selectedLayout = i
					m.activeView = viewDirBrowse
					m.dirList.Title = "Browse ❯ " + m.currentBrowseDir

					return m, readDir(m.currentBrowseDir, m.currentBrowseDir == m.workspaceDir)
				}

			// Step 2: User selects a directory
			case viewDirBrowse:
				if i, ok := m.dirList.SelectedItem().(genericItem); ok {
					switch i.title {

					case "[Select This Directory]":
						m.targetDir = m.currentBrowseDir
						// default to directory name for tmux session name
						m.sessionName = filepath.Base(m.currentBrowseDir)
						return m, tea.Quit

					case "[Create New Project Here]":
						m.activeView = viewNameInput
						return m, m.textInput.Focus()

					case "[Go Up One Level]":
						m.currentBrowseDir = filepath.Dir(m.currentBrowseDir)
						m.dirList.Title = "Browse ❯ " + m.currentBrowseDir
						// Re-scan the parent directory
						return m, readDir(m.currentBrowseDir, m.currentBrowseDir == m.workspaceDir)

					default:
						// User selected a standard directory
						m.currentBrowseDir = filepath.Join(m.currentBrowseDir, i.title)
						m.dirList.Title = "Browse ❯ " + m.currentBrowseDir
						// Re-scan newly entered directory
						return m, readDir(m.currentBrowseDir, m.currentBrowseDir == m.workspaceDir)
					}
				}

			// Step 3: User confirms a new project name
			case viewNameInput:
				if m.textInput.Value() != "" {
					m.targetDir = filepath.Join(m.currentBrowseDir, m.textInput.Value())
					m.sessionName = m.textInput.Value()

					os.MkdirAll(m.targetDir, 0755)

					return m, tea.Quit
				}
			}
		case "esc":
			switch m.activeView {
			case viewDirBrowse:
				m.activeView = viewSessionSelect
			case viewNameInput:
				m.activeView = viewDirBrowse
			}
		}
	}

	switch m.activeView {
	case viewSessionSelect:
		m.sessionList, cmd = m.sessionList.Update(msg)
	case viewDirBrowse:
		m.dirList, cmd = m.dirList.Update(msg)
	case viewNameInput:
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	var content string

	switch m.activeView {
	case viewSessionSelect:
		content = m.sessionList.View()
	case viewDirBrowse:
		content = m.dirList.View()
	case viewNameInput:
		content = m.textInput.View()
	default:
		content = "Unknown state"
	}

	v := tea.NewView(content)

	v.AltScreen = true

	return v
}

func initialModel() model {
	// 1. Resolve base paths
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/root" // safe fallback
	}
	workspace := filepath.Join(home, "workspace")

	//! FIXME: Handle this gracefully
	// 2. Build the static session options
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config Error: %v\n", err)
	}

	items := make([]list.Item, len(cfg.Sessions))
	for i, session := range cfg.Sessions {
		items[i] = session
	}

	sessionList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	sessionList.Title = "Select Session ❯"
	sessionList.SetShowStatusBar(false)
	sessionList.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Placeholder = "untitled_project"
	ti.Focus()
	ti.CharLimit = 64
	ti.SetWidth(40)

	dirList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	dirList.Title = "Select Project ❯"
	dirList.SetShowStatusBar(true)

	return model{
		activeView:       viewSessionSelect,
		workspaceDir:     workspace,
		currentBrowseDir: workspace,
		sessionList:      sessionList,
		dirList:          dirList,
		textInput:        ti,
		editor:           cfg.Settings.Editor,
	}
}

func buildTmuxCommand(fm model) string {
	targetDir := shellQuote(fm.targetDir)
	sessionName := shellQuote(fm.sessionName)
	editor := fm.editor

	var cmds []string

	cmds = append(cmds, fmt.Sprintf("cd %s", targetDir))

	for wIdx, window := range fm.selectedLayout.Windows {
		winName := shellQuote(window.Name)

		if wIdx == 0 {
			cmds = append(cmds, fmt.Sprintf("tmux new-session -d -s %s -n %s -c %s", sessionName, winName, targetDir))
		} else {
			cmds = append(cmds, fmt.Sprintf("tmux new-window -t %s -n %s -c %s", sessionName, winName, targetDir))
		}

		for pIdx, pane := range window.Panes {
			paneTarget := fmt.Sprintf("%s:%d.%d", sessionName, wIdx, pIdx)

			if pIdx > 0 {
				splitFlag := "-v"
				if pane.Split == "vertical" {
					splitFlag = "-h"
				}

				sizeArg := ""
				if pane.Size != "" {
					sizeArg = fmt.Sprintf("-l %s", shellQuote(pane.Size))
				}

				prevPaneTarget := fmt.Sprintf("%s:%d.%d", sessionName, wIdx, (pIdx - 1))
				cmds = append(cmds, fmt.Sprintf("tmux split-window %s %s -t %s -c %s", splitFlag, sizeArg, prevPaneTarget, targetDir))
			}

			if pane.Command != "" {
				rawCmd := strings.ReplaceAll(pane.Command, "{{editor}}", editor)

				cmds = append(cmds, fmt.Sprintf("tmux send-keys -t %s %s C-m", paneTarget, shellQuote(rawCmd)))
			}
		}
	}

	cmds = append(cmds, fmt.Sprintf("tmux select-pane -t %s:0.0", sessionName))
	cmds = append(cmds, fmt.Sprintf("tmux attach-session -t %s", sessionName))

	return strings.Join(cmds, " && ")
}

func printBashWrapper() {
	wrapper := `
# Yggdrasil Shell Hook
# Do not edit this manually. Generated by 'yggdrasil init bash'

ygg() {
  local output
  # Run the Yggdrasil binary and capture stdout
  output = "$(yggdrasil "$@")"

  # Evaluate the returned commands in the current parent shell
  if [-n "$output" ]; then
    eval "$output"
  fi
}
`
	fmt.Print(wrapper)
}

func handleCLI() {
	if len(os.Args) < 2 {
		return
	}

	switch os.Args[1] {
	case "init":
		if len(os.Args) > 2 && os.Args[2] == "bash" {
			printBashWrapper()
			os.Exit(0)
		}

		fmt.Fprintln(os.Stderr, "Usage: yggdrasil init bash")
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func main() {
	handleCLI()

	m := initialModel()

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running Yggdrasil: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := finalModel.(model); ok {
		// Outputs a raw bash command to stdout to be executed by shell wrapper
		if fm.targetDir != "" && fm.sessionName != "" {
			fmt.Println(buildTmuxCommand(fm))
		}
	}
}
