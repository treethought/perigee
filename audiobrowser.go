package main

import (
	"fmt"
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

func (a *audioFile) Row(ref string) table.Row {
	// todo pass in cols to truncate based on cols widths
	return table.Row{
		ref,
		a.name,
		a.path,
	}
}

type audioBrowserKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
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
	Select: key.NewBinding(
		key.WithKeys("enter", "space"),
		key.WithHelp("enter/space", "play sample"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace", "h", "left"),
		key.WithHelp("backspace", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("ctrl+w", "exit filter or browser"),
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
	onSelect func(path string) tea.Cmd

	fi         textinput.Model
	filtering  bool
	query      string
	setRows    []table.Row
	sampleRows map[string][]table.Row
	currentSet string

	// allRows      []table.Row
	filteredRows []table.Row
	active       bool
}

func NewAudioBrowser() *AudioBrowser {
	columns := []table.Column{
		{Title: "Ref", Width: 30},
		{Title: "Name", Width: 10},
		// {Title: "Size", Width: 10},
		{Title: "", Width: 0},
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

	cols := m.t.Columns()

	if width < 32 {
		cols[0].Width = width
	} else {
		cols[0].Width = width / 2
		cols[1].Width = width / 2
		cols[2].Width = 0
	}
	m.t.SetColumns(cols)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func (m *AudioBrowser) SetOnSelect(f func(path string) tea.Cmd) {
	m.onSelect = f
}

func (m *AudioBrowser) applyFilter() {
	all := m.setRows
	if m.currentSet != "" {
		all = m.sampleRows[m.currentSet]
	}
	q := strings.ToLower(m.fi.Value())
	if q == "" {
		m.filteredRows = all
		m.t.SetRows(all)
		return
	}

	var filtered []table.Row
	for _, row := range all {
		if strings.Contains(strings.ToLower(row[0]), q) {
			filtered = append(filtered, row)
		}
	}
	m.filteredRows = filtered
	m.t.SetRows(filtered)
}

func (m *AudioBrowser) SetFiles(files map[string][]audioFile) tea.Cmd {
	var sampleSetRows []table.Row
	sampleRows := make(map[string][]table.Row)

	for set, samples := range files {
		sampleSetRows = append(sampleSetRows, table.Row{
			truncate(set, 30), "", fmt.Sprintf("%d", len(files[set])),
		})
		sampleRows[set] = make([]table.Row, len(samples))
		for i, sample := range samples {
			ref := fmt.Sprintf("%s:%d", set, i)
			sampleRows[set][i] = sample.Row(ref)
		}
	}

	sort.Slice(sampleSetRows, func(i, j int) bool {
		return strings.ToLower(sampleSetRows[i][0]) < strings.ToLower(sampleSetRows[j][0])
	})

	m.setRows = sampleSetRows
	m.sampleRows = sampleRows
	all := m.setRows
	if m.currentSet != "" {
		all = m.sampleRows[m.currentSet]
	}
	m.filteredRows = all
	m.t.SetRows(all)
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

		case key.Matches(msg, defaultAudioBrowserKeyMap.Back):
			if m.currentSet != "" {
				m.currentSet = ""
				m.t.SetRows(m.setRows)
				return m, nil
			}
			m.active = false
			return m, nil

		case key.Matches(msg, defaultAudioBrowserKeyMap.Select):
			if m.currentSet == "" {
				sampleSet := m.t.SelectedRow()[0]
				m.currentSet = sampleSet
				m.t.SetRows(m.sampleRows[m.currentSet])
				return m, nil
			}
			row := m.t.SelectedRow()
			path := row[2]

			if m.onSelect != nil {
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
		Render(fmt.Sprint("Sample Browser"))

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
