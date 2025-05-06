package main

import (
	"log"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
)

func renderDialog(text string, w, h int) string {
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(text),
		// lipgloss.WithWhitespaceChars("猫咪"),
		lipgloss.WithWhitespaceForeground(subtle),
	)
}

type QuickSelect struct {
	active   bool
	l        *list.Model
	w, h     int
	onSelect func(i *selectItem) tea.Cmd
}

type selectItem struct {
	name  string
	value string
}

func (m *selectItem) FilterValue() string {
	return m.name
}
func (m *selectItem) View() string {
	return m.value
}
func (m *selectItem) Title() string {
	return m.name
}

func (i *selectItem) Description() string {
	return ""
}

func NewQuickSelect() *QuickSelect {
	d := list.NewDefaultDelegate()
	d.SetHeight(1)
	d.ShowDescription = true

	l := list.New([]list.Item{}, d, 0, 0)
	l.KeyMap.CursorUp.SetKeys("k", "up")
	l.KeyMap.CursorDown.SetKeys("j", "down")
	l.KeyMap.Quit.SetKeys("ctrl+c")

	l.Title = "View Console"
	l.SetShowTitle(true)
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowHelp(true)
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)

	return &QuickSelect{l: &l}
}

func (m *QuickSelect) SetOnSelect(f func(i *selectItem) tea.Cmd) {
	m.onSelect = f
}

func (m *QuickSelect) SetSize(w, h int) {
	m.w = w
	m.h = h
	m.l.SetSize(w, h)
	log.Println("quick select size:", w, h)
}
func (m *QuickSelect) Active() bool {
	return m.active
}
func (m *QuickSelect) SetActive(active bool) {
	m.active = active
}

func (m *QuickSelect) Init() tea.Cmd {
	items := []list.Item{
		&selectItem{name: "Tidal Repl", value: "tidalrepl"},
		&selectItem{name: "SCLang", value: "sclang"},
	}
	return m.l.SetItems(items)
}

func (m *QuickSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.active {
			return m, nil
		}
		if msg.String() == "enter" {
			if m.onSelect != nil {
				return m, m.onSelect(m.l.SelectedItem().(*selectItem))
			}
			currentItem, ok := m.l.SelectedItem().(*selectItem)
			if !ok {
				log.Println("no item selected")
				return m, nil
			}
			log.Println("selected item:", currentItem.name)
		}
	}
	l, cmd := m.l.Update(msg)
	m.l = &l
	return m, cmd
}

func (m *QuickSelect) View() string {
	if !m.active {
		return ""
	}
	return renderDialog(m.l.View(), m.w, m.h)
}
