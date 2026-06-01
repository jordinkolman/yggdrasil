package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v3"
)

var (
  vikingOrange = lipgloss.Color("#FF8C00")
  highlight    = lipgloss.Color("#FFA500")
  textMain     = lipgloss.Color("#EEEEEE")
  textSubtle   = lipgloss.Color("#555555")
  borderSubtle = lipgloss.Color("#333333")

  appStyle = lipgloss.NewStyle().Margin(1, 2)

  headerStyle = lipgloss.NewStyle().
          Foreground(lipgloss.Color("#000000")).
          Background(vikingOrange).
          Padding(0, 2).
          Bold(true).
          MarginBottom(1)

  listTitleStyle = lipgloss.NewStyle().
          Foreground(vikingOrange).
          Bold(true).
          Padding(0, 0, 1, 2)

  panelStyle = lipgloss.NewStyle().
          Border(lipgloss.RoundedBorder()).
          BorderForeground(borderSubtle).
          Padding(1, 2)

  footerStyle = lipgloss.NewStyle().
          Foreground(textSubtle).
          MarginTop(1)

  inputBoxStyle = lipgloss.NewStyle().
          Border(lipgloss.RoundedBorder()).
          BorderForeground(vikingOrange).
          Padding(1).        
          MarginTop(1)
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
  path  string
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
	sessionList    list.Model
	dirList        list.Model
	textInput      textinput.Model
  previewContent string

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
							{Split: "", Size: "", Command: ""}, // Empty command drops to default shell
							{Split: "vertical", Size: "33%", Command: ""},
							{Split: "horizontal", Size: "50%", Command: ""},
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
              {Split: "", Size: "", Command: "{{editor}} ."},
							{Split: "vertical", Size: "30%", Command: ""},
							{Split: "horizontal", Size: "66%", Command: ""},
							{Split: "horizontal", Size: "50%", Command: ""},
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

	configDir := filepath.Join(home, ".config", "yggdrasil")
	configPath := filepath.Join(configDir, "config.yaml")

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

      fullPath := filepath.Join(targetPath, name)
			items = append(items, genericItem{
				title: name,
				desc:  fullPath,
        path:  fullPath,
			})
		}

		return dirScanMsg(items)
	}
}

func generatePreview(path string) string {
  if path == "" {
    return lipgloss.NewStyle().Foreground(textSubtle).Render("No preview available")
  }

  info, err := os.Stat(path)
  if err != nil || !info.IsDir() {
    return lipgloss.NewStyle().Foreground(textSubtle).Render("Invalid directory")
  }

  entries, err := os.ReadDir(path)
	if err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("Permission Denied.")
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(highlight).Render(filepath.Base(path)) + "\n\n")

	if len(entries) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(textSubtle).Render("Directory is empty."))
		return sb.String()
	}

	// Show up to 15 items in the preview
	displayLimit := 15
	for i, e := range entries {
		if i >= displayLimit {
			sb.WriteString(lipgloss.NewStyle().Foreground(textSubtle).Render(fmt.Sprintf("\n... and %d more items", len(entries)-displayLimit)))
			break
		}

		icon := "📄 "
		if e.IsDir() {
			icon = "📁 "
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", icon, e.Name()))
	}

	return sb.String()
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
    appW, appH := appStyle.GetFrameSize()

    panelW, panelH := panelStyle.GetFrameSize()

    headerAndFooterHeight := 4

		m.width = msg.Width - appW - panelW
		m.height = msg.Height - appH - panelH - headerAndFooterHeight

		m.sessionList.SetSize(m.width, m.height)
		m.dirList.SetSize(m.width/2, m.height)
		return m, nil

	case dirScanMsg:
		cmd = m.dirList.SetItems(msg)
    m.dirList.Select(0)

    if i, ok := m.dirList.SelectedItem().(genericItem); ok {
      m.previewContent = generatePreview(i.path)
    }
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
    cmds = append(cmds, cmd)

	case viewDirBrowse:
    oldIndex := m.dirList.Index()
		m.dirList, cmd = m.dirList.Update(msg)
    cmds = append(cmds, cmd)

  if m.dirList.Index() != oldIndex {
      if i, ok := m.dirList.SelectedItem().(genericItem); ok {
        m.previewContent = generatePreview(i.path)
      }
    }
	case viewNameInput:
		m.textInput, cmd = m.textInput.Update(msg)
    cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	var activeContent string
	var headerTitle string

	// 1. Determine what the main content and header should be
	switch m.activeView {
	case viewSessionSelect:
		headerTitle = " YGGDRASIL | Select Layout "
		activeContent = panelStyle.Render(m.sessionList.View())

	case viewDirBrowse:
		headerTitle = " YGGDRASIL | Target Directory "
		
		// Define the style for the right-hand preview pane
		previewPaneStyle := lipgloss.NewStyle().
			Width(m.width / 2).        // Consume the remaining 50% of the space
			Height(m.height).          // Match the list height
			Border(lipgloss.NormalBorder(), false, false, false, true). // Left border only
			BorderForeground(borderSubtle).
			Padding(0, 1, 0, 2)
			
		// Render the preview text inside the pane
		previewPane := previewPaneStyle.Render(m.previewContent)

		
		// Render the list
		listPane := m.dirList.View()
		
		// Stitch them together horizontally
		combinedView := lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)
		
		activeContent = panelStyle.Render(combinedView)
	case viewNameInput:
		headerTitle = " YGGDRASIL | Initialize Project "
		inputPrompt := lipgloss.NewStyle().Foreground(textMain).Render("Project Name:")
		inputBox := inputBoxStyle.Render(m.textInput.View())
		
		rawContent := lipgloss.JoinVertical(lipgloss.Left, inputPrompt, inputBox)
		activeContent = panelStyle.Render(rawContent)

	default:
		activeContent = "Unknown state"
	}

	// 2. Render Layout Components
	header := headerStyle.Render(headerTitle)
	
	// A simple footer displaying the global editor setting
	footerText := fmt.Sprintf("Editor: %s • ctrl+c: quit • esc: back", m.editor)
	footer := footerStyle.Render(footerText)

	// 3. Assemble the App
	// We stack Header -> Content -> Footer
	ui := lipgloss.JoinVertical(lipgloss.Left,
		header,
		activeContent,
		footer,
	)

	// Apply the outer margin
	finalRender := appStyle.Render(ui)

	v := tea.NewView(finalRender)
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

	// Build a custom delegate for a premium feel
	delegate := list.NewDefaultDelegate()
	
	// Active Item: Viking Orange with a thick left border block
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(vikingOrange).
		Foreground(vikingOrange).
		Padding(0, 0, 0, 1).
		Bold(true)
		
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(vikingOrange).
		Foreground(highlight).
		Padding(0, 0, 0, 1)

	// Inactive Items: Dimmed out so they fade into the background
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(textMain).
		Padding(0, 0, 0, 2)
		
	delegate.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(textSubtle).
		Padding(0, 0, 0, 2)


	sessionList := list.New(items, delegate, 0, 0)
  sessionList.Title = "Select Session ❯"
  sessionList.Styles.Title = listTitleStyle
	sessionList.SetShowStatusBar(false)
	sessionList.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Placeholder = "untitled_project"
	ti.Focus()
	ti.CharLimit = 64
	ti.SetWidth(40)

  inputStyles := ti.Styles()
  inputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(vikingOrange)
  inputStyles.Focused.Text = lipgloss.NewStyle().Foreground(vikingOrange)
  ti.SetStyles(inputStyles)

	dirList := list.New([]list.Item{}, delegate, 0, 0)
  dirList.Title = "Select Project ❯"
  dirList.Styles.Title = listTitleStyle
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
  output="$(yggdrasil "$@")"

  # Evaluate the returned commands in the current parent shell
  if [ -n "$output" ]; then
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
