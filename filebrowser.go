package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fileItem struct {
	path     string
	name     string
	isDir    bool
	selected bool
}

func (i fileItem) Title() string {
	if i.isDir {
		return fmt.Sprintf("📁 %s", i.name)
	}
	return fmt.Sprintf("📄 %s", i.name)
}

func (i fileItem) Description() string {
	return i.path
}

func (i fileItem) FilterValue() string {
	return i.name
}

type FileBrowser struct {
	l        list.Model
	active   bool
	curDir   string
	onSelect func(path string) tea.Cmd
}

type fileBrowserKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	GoToRoot key.Binding
}

var defaultFileBrowserKeyMap = fileBrowserKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open file/directory"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace", "h", "left"),
		key.WithHelp("backspace", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "quit"),
	),
	GoToRoot: key.NewBinding(
		key.WithKeys("g", "/"),
		key.WithHelp("g", "go to root"),
	),
}

func NewFileBrowser() *FileBrowser {
	// Start in the current directory
	curDir, err := os.Getwd()
	if err != nil {
		curDir = "."
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#FFFF00")).Background(lipgloss.Color("#333333"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#FFFF00")).Background(lipgloss.Color("#333333"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "File Browser"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)

	m := &FileBrowser{
		l:      l,
		curDir: curDir,
	}

	return m
}

func (m *FileBrowser) SetSize(width, height int) {
	m.l.SetSize(width, height)
}

func (m *FileBrowser) SetActive(active bool) {
	m.active = active
}

func (m *FileBrowser) Active() bool {
	return m.active
}

func (m *FileBrowser) SetOnSelect(f func(path string) tea.Cmd) {
	m.onSelect = f
}

func (m *FileBrowser) loadFiles() tea.Cmd {
	return func() tea.Msg {
		var items []list.Item

		entries, err := os.ReadDir(m.curDir)
		if err != nil {
			log.Println("Error reading directory:", err)
			return nil
		}

		// Sort entries: directories first, then files, all alphabetically
		sort.Slice(entries, func(i, j int) bool {
			iIsDir := entries[i].IsDir()
			jIsDir := entries[j].IsDir()
			if iIsDir && !jIsDir {
				return true
			}
			if !iIsDir && jIsDir {
				return false
			}
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		})

		// Add entries to list
		for _, entry := range entries {
			// Skip hidden files and directories
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			_, err := entry.Info()
			if err != nil {
				continue
			}

			items = append(items, fileItem{
				path:  filepath.Join(m.curDir, entry.Name()),
				name:  entry.Name(),
				isDir: entry.IsDir(),
			})
		}

		return m.l.SetItems(items)
	}
}

func (m *FileBrowser) SetDirectory(path string) tea.Cmd {
	m.curDir = path
	return m.loadFiles()
}

func (m *FileBrowser) Init() tea.Cmd {
	return m.loadFiles()
}

func (m *FileBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keyboard input
		switch {
		case key.Matches(msg, defaultFileBrowserKeyMap.Quit):
			m.active = false
			return m, nil
		case key.Matches(msg, defaultFileBrowserKeyMap.Enter):
			selectedItem := m.l.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}

			fileItem := selectedItem.(fileItem)
			if fileItem.isDir {
				return m, m.SetDirectory(fileItem.path)
			} else if m.onSelect != nil {
				return m, m.onSelect(fileItem.path)
			}
		case key.Matches(msg, defaultFileBrowserKeyMap.Back):
			// Go up one directory
			if m.curDir != "/" {
				m.SetDirectory(filepath.Dir(m.curDir))
			}
			return m, nil
		case key.Matches(msg, defaultFileBrowserKeyMap.GoToRoot):
			// Go to root directory
			m.SetDirectory("/")
			return m, nil
		}

		var cmd tea.Cmd
		m.l, cmd = m.l.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.l, cmd = m.l.Update(msg)
	return m, cmd
}

func (m *FileBrowser) View() string {
	if !m.active {
		return ""
	}

	title := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Width(m.l.Width()).
		Padding(0, 1).
		Render(fmt.Sprintf("File Browser - %s", m.curDir))

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		m.l.View(),
	)
}
