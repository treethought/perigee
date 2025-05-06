package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/vimtea"
)

type consoleMsg string
type sclangMsg string

func listenSclang(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return sclangMsg(<-ch)
	}
}

func listenConsole(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return consoleMsg(<-ch)
	}
}

func sclangStartCmd(sclang *SCLangRepl) tea.Cmd {
	return func() tea.Msg {
		return sclang.Start()
	}
}

func replStartCmd(repl *TidalRepl) tea.Cmd {
	return func() tea.Msg {
		return repl.Start()
	}
}

type keyMap struct {
	Quit             key.Binding
	FocusEditor      key.Binding
	FocusConsole     key.Binding
	FocusQuickSelect key.Binding
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
	FocusQuickSelect: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open quick select"),
	),
}

type App struct {
	editor        *Editor
	repl          *TidalRepl
	console       *Console
	sclang        *SCLangRepl
	scConsole     *Console
	qs            *QuickSelect
	active        tea.Model
	activeConsole *Console
}

func NewApp(repl *TidalRepl, sclang *SCLangRepl) *App {
	return &App{
		repl:      repl,
		sclang:    sclang,
		console:   NewConsole(0, 10),
		scConsole: NewConsole(10, 0),
		editor:    NewEditor(repl.Send),
		qs:        NewQuickSelect(),
	}
}

func (a *App) SetActive(m tea.Model) {
	a.active = m
}

func (a *App) setActiveConsole(val string) {
	switch val {
	case "sclang":
		a.activeConsole = a.scConsole
	case "console":
		a.activeConsole = a.console
	default:
		a.activeConsole = a.console
	}
}

func (a *App) Init() tea.Cmd {
	a.activeConsole = a.console
	a.SetActive(a.editor)
	a.editor.e.SetMode(vimtea.ModeInsert)
	a.qs.SetOnSelect(func(item *selectItem) tea.Cmd {
		if item == nil {
			return nil
		}
		a.setActiveConsole(item.value)
		a.qs.SetActive(false)
		return nil
	})

	return tea.Batch(
		a.qs.Init(),
		replStartCmd(a.repl),
		sclangStartCmd(a.sclang),
		listenConsole(a.repl.out),
		listenSclang(a.sclang.out),
		a.editor.load("perigee.tidal"),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.qs.SetSize(msg.Width/2, msg.Height/2)

		ch := msg.Height / 8
		if ch < 10 {
			ch = 10
		}
		a.console.SetSize(msg.Width, ch)
		a.scConsole.SetSize(msg.Width, ch)

		_, cmd := a.editor.SetSize(msg.Width, msg.Height-(ch)) // Reserve space for console
		return a, cmd

	case consoleMsg:
		a.console.AddLine(string(msg))
		return a, listenConsole(a.repl.out)

	case sclangMsg:
		a.scConsole.AddLine(string(msg))
		return a, listenSclang(a.sclang.out)

	case tea.KeyMsg:
		if a.active == nil {
			a.active = a.editor
		}

		if a.active == a.editor && !(a.editor.e.GetMode() == vimtea.ModeNormal) {
			_, cmd := a.editor.Update(msg)
			return a, cmd
		}

		switch {
		case key.Matches(msg, defaultKeyMap.FocusQuickSelect):
			a.qs.SetActive(true)
			a.active = a.qs
			return a, nil
		case key.Matches(msg, defaultKeyMap.Quit):
			return a, tea.Quit
		case key.Matches(msg, defaultKeyMap.FocusEditor):
			a.active = a.editor
			return a, a.editor.e.SetStatusMessage("")
		case key.Matches(msg, defaultKeyMap.FocusConsole):
			a.active = a.activeConsole
			return a, a.editor.e.SetStatusMessage("console focused")
		}
	}
	_, cmd := a.active.Update(msg)
	if a.activeConsole != nil {
		_, ccmd := a.activeConsole.Update(msg)
		return a, tea.Batch(cmd, ccmd)
	}
	return a, cmd
}

func (a *App) View() string {
	if a.qs.Active() {
		return a.qs.View()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		a.editor.e.View(),
		a.activeConsole.View(),
	)
}
