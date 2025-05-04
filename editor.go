package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kujtimiihoxha/vimtea"
)

type sendFunc func(s string) error

type Editor struct {
	e    vimtea.Editor
	send sendFunc
}

func NewEditor(send sendFunc) *Editor {
	m := &Editor{
		send: send,
		e:    vimtea.NewEditor(vimtea.WithFileName("tidal.hs")),
	}

	m.e.AddCommand("q", func(b vimtea.Buffer, a []string) tea.Cmd {
		return func() tea.Msg {
			return tea.Quit()
		}
	})

	m.e.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+e",
		Mode:        vimtea.ModeNormal,
		Description: "Send buffer to tidal",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			if err := m.send(b.Text()); err != nil {
				return vimtea.SetStatusMsg(fmt.Sprintf("Error sending command: %v", err))
			}
			return vimtea.SetStatusMsg("sent!")
		},
	})

	m.e.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+h",
		Mode:        vimtea.ModeNormal,
		Description: "Hush",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			if err := m.send("hush"); err != nil {
				return vimtea.SetStatusMsg(fmt.Sprintf("Error sending command: %v", err))
			}
			return vimtea.SetStatusMsg("Hushed!")
		},
	})

	m.e.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+s",
		Mode:        vimtea.ModeNormal,
		Description: "Save file",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			return vimtea.SetStatusMsg("File saved!")
		},
	})
	return m
}

func (m *Editor) SetSize(width, height int) (vimtea.Editor, tea.Cmd) {
	ed, cmd := m.e.SetSize(width, height)
	m.e = ed.(vimtea.Editor)
	return m.e, cmd
}

func (m *Editor) Init() tea.Cmd {
	return m.e.Init()
}
func (m *Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.e.Update(msg)
	if model != nil {
		m.e = model.(vimtea.Editor)
	}
	return m, cmd
}

func (m *Editor) View() string {
	return m.e.View()
}
