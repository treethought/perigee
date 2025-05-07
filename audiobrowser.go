package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type noOp struct{}

type audioFile struct {
	path     string
	name     string
	fileType string
	size     string
	selected bool
}

type audioBrowserKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Play     key.Binding
	Back     key.Binding
	Quit     key.Binding
	GoToRoot key.Binding
	Filter   key.Binding
}

var defaultAudioBrowserKeyMap = audioBrowserKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Play: key.NewBinding(
		key.WithKeys("enter", "space"),
		key.WithHelp("enter/space", "play sample"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace", "h", "left"),
		key.WithHelp("backspace", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "exit filter or browwer"),
	),
	GoToRoot: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "go to root"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
}

type AudioBrowser struct {
	t        table.Model
	curDir   string
	onSelect func(path string) tea.Cmd

	fi           textinput.Model
	filtering    bool
	query        string
	allRows      []table.Row
	filteredRows []table.Row
	active       bool
}

func NewAudioBrowser() *AudioBrowser {
	columns := []table.Column{
		{Title: "Ref", Width: 30},
		{Title: "Type", Width: 10},
		{Title: "Size", Width: 10},
		{Title: "Name", Width: 10},
		{Title: "Path", Width: 0},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	// Set default styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFF00")).
		Background(lipgloss.Color("#333333")).
		Bold(true)

	t.SetStyles(s)

	fi := textinput.New()
	fi.Placeholder = "Filter"
	fi.Focus()
	fi.Width = 20
	fi.Prompt = "🔍 "
	fi.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	m := &AudioBrowser{
		t:  t,
		fi: fi,
	}

	return m
}

func (m *AudioBrowser) SetActive(active bool) {
	m.active = active
}
func (m *AudioBrowser) Active() bool {
	return m.active
}

func (m *AudioBrowser) SetSize(width, height int) {
	m.t.SetWidth(width)
	m.t.SetHeight(height - 2)
	m.fi.Width = width - 5 // Adjust text input width
}

func (m *AudioBrowser) SetOnSelect(f func(path string) tea.Cmd) {
	m.onSelect = f
}

func (m *AudioBrowser) applyFilter() {
	q := strings.ToLower(m.fi.Value())
	if q == "" {
		m.filteredRows = m.allRows
		m.t.SetRows(m.allRows)
		return
	}

	var filtered []table.Row
	for _, row := range m.allRows {
		if strings.Contains(strings.ToLower(row[0]), q) {
			filtered = append(filtered, row)
		}
	}
	m.filteredRows = filtered
	m.t.SetRows(filtered)
}

func (m *AudioBrowser) SetFiles(files map[string][]audioFile) tea.Cmd {
	var rows []table.Row

	// Sort entries: directories first, then files, all alphabetically
	for _, samples := range files {
		sort.Slice(samples, func(i, j int) bool {
			return strings.ToLower(samples[i].name) < strings.ToLower(samples[j].name)
		})
	}

	for set, samples := range files {
		for i, f := range samples {
			if strings.HasPrefix(f.name, ".") {
				continue
			}

			if _, ok := audioExts[filepath.Ext(f.path)]; !ok {
				continue
			}
			ref := fmt.Sprintf("%s:%d", set, i)

			rows = append(rows, table.Row{
				ref,
				f.fileType,
				f.size,
				f.name,
				f.path,
			})
		}
	}
	m.allRows = rows
	m.filteredRows = rows
	m.t.SetRows(rows)
	m.t.KeyMap.PageDown.SetKeys("ctrl+f", "pgdn")
	return nil
}

func (m *AudioBrowser) Init() tea.Cmd {
	return nil
}

func (m *AudioBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keyboard input
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.fi.Blur()
				return m, nil
			case "enter":
				m.filtering = false
				m.applyFilter()
				return m, nil
			default:
				fi, cmd := m.fi.Update(msg)
				m.fi = fi
				m.applyFilter()
				return m, cmd

			}
		}
		switch {
		case key.Matches(msg, defaultAudioBrowserKeyMap.Filter):
			m.filtering = true
			return m, m.fi.Focus()
		case key.Matches(msg, defaultAudioBrowserKeyMap.Quit):
			m.active = false
			return m, nil

		case key.Matches(msg, defaultAudioBrowserKeyMap.Play):
			row := m.t.SelectedRow()

			path := row[4]
			fileType := row[1]

			if fileType == "DIR" {
				log.Println("Selected directory:", path)
				return m, nil
			} else if m.onSelect != nil {
				return m, m.onSelect(path)
			}
		}
	}

	var cmd tea.Cmd
	m.t, cmd = m.t.Update(msg)
	return m, cmd
}

func (m *AudioBrowser) View() string {
	if !m.active {
		return ""
	}
	title := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("Audio Browser - %s", m.curDir))

	var fview string
	if m.filtering {
		fview = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Width(m.t.Width()).
			Render(m.fi.View())
	} else if m.fi.Value() != "" {
		fview = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Width(m.t.Width()).
			Render(fmt.Sprintf("Filter: %s", m.fi.Value()))
	} else {
		fview = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#999999")).
			Padding(0, 1).
			Width(m.t.Width()).
			Render("Press '/' to filter")
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		fview,
		m.t.View(),
	)
}
