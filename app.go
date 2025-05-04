package main

import (
	"log"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/vimtea"
)

type consoleMsg string

func listenConsole(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return consoleMsg(<-ch)
	}
}

func replStartCmd(repl *TidalRepl) tea.Cmd {
	return func() tea.Msg {
		return repl.Start()
	}
}

type keyMap struct {
	Quit         key.Binding
	FocusEditor  key.Binding
	FocusConsole key.Binding
}

var defaultKeyMap = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c", "quit"),
	),
	FocusEditor: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "switch focus back to editor"),
	),
	FocusConsole: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "focus console"),
	),
}

type App struct {
	editor  *Editor
	repl    *TidalRepl
	console *Console
	active  tea.Model
}

func NewApp(repl *TidalRepl) *App {
	return &App{
		repl:    repl,
		console: NewConsole(0, 10),
		editor:  NewEditor(repl.Send),
	}
}

func (a *App) SetActive(m tea.Model) {
	a.active = m
}

func (a *App) Init() tea.Cmd {
	a.SetActive(a.editor)
	a.editor.e.SetMode(vimtea.ModeInsert)
	return tea.Batch(
		replStartCmd(a.repl),
		listenConsole(a.repl.out),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ch := msg.Height / 4
		if ch < 10 {
			ch = 10
		}
		a.console.SetSize(msg.Width, ch)
		_, cmd := a.editor.SetSize(msg.Width, msg.Height-ch) // Reserve space for console
		return a, cmd

	case consoleMsg:
		a.console.AddLine(string(msg))
		return a, listenConsole(a.repl.out)

	case tea.KeyMsg:
		if a.active == nil {
			a.active = a.editor
		}

		if a.active == a.editor && !(a.editor.e.GetMode() == vimtea.ModeNormal) {
			_, cmd := a.editor.Update(msg)
			return a, cmd
		}

		switch {
		case key.Matches(msg, defaultKeyMap.Quit):
			return a, tea.Quit
		case key.Matches(msg, defaultKeyMap.FocusEditor):
			log.Print("switching focus to editor")
			a.active = a.editor
			return a, a.editor.e.SetStatusMessage("")
		case key.Matches(msg, defaultKeyMap.FocusConsole):
			log.Print("switching focus to console")
			a.active = a.console
			return a, a.editor.e.SetStatusMessage("console focused")
		}
	}
	_, cmd := a.active.Update(msg)
	return a, cmd
}

func (a *App) View() string {

	return lipgloss.JoinVertical(
		lipgloss.Left,
		a.editor.e.View(),
		a.console.View(),
	)
}
