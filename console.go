package main

import (
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Console struct {
	Lines    []string
	viewport viewport.Model
}

func NewConsole(width, height int) *Console {
	return &Console{
		Lines:    make([]string, 0),
		viewport: viewport.New(width, height),
	}
}

func (c *Console) SetSize(width, height int) {
	log.Printf("Setting console size to %dx%d", width, height)
	c.viewport.Width = width
	c.viewport.Height = height
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
	return c.viewport.View()
}
