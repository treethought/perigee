package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	consoleStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), true)
)

type Console struct {
	Lines    []string
	viewport viewport.Model
	active   bool
}

func NewConsole(width, height int) *Console {
	return &Console{
		Lines:    make([]string, 0),
		viewport: viewport.New(width, height),
	}
}

func (c *Console) Active() bool {
	return c.active
}

func (c *Console) SetActive(active bool) {
	c.active = active
}

func (c *Console) SetSize(width, height int) {
	sx, sy := consoleStyle.GetFrameSize()
	c.viewport.Width = width - sx
	c.viewport.Height = height - sy
	c.viewport.SetContent(strings.Join(c.Lines, "\n"))
	c.viewport.GotoBottom()
}

func (c *Console) AddLine(line string) {
	c.Lines = append(c.Lines, line)
	c.viewport.SetContent(strings.Join(c.Lines, "\n"))
	c.viewport.GotoBottom()
}

func (c *Console) Init() tea.Cmd {
	return nil
}

func (c *Console) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	v, cmd := c.viewport.Update(msg)
	c.viewport = v
	return c, cmd
}

func (c *Console) View() string {
  if !c.active {
    return ""
  }
	return consoleStyle.Render(c.viewport.View())
}
