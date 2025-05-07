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
	Quit              key.Binding
	FocusEditor       key.Binding
	FocusConsole      key.Binding
	FocusQuickSelect  key.Binding
	FocusFileBrowser  key.Binding
	FocusAudioBrowser key.Binding
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
	FocusFileBrowser: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "open file browser"),
	),
	FocusAudioBrowser: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("ctrl+w", "open audio browser"),
	),
}

type App struct {
	cfg           *Config
	editor        *Editor
	repl          *TidalRepl
	console       *Console
	sclang        *SCLangRepl
	scConsole     *Console
	qs            *QuickSelect
	fileBrowser   *FileBrowser
	sampleBrowser *SampleBrowser
	active        tea.Model
	activeConsole *Console
}

func NewApp(cfg *Config) *App {
	repl := NewTidalRepl(cfg.Bootfile)
	sclang := NewSCLangRepl("")
	return &App{
		cfg:           cfg,
		repl:          repl,
		sclang:        sclang,
		console:       NewConsole(0, 10),
		scConsole:     NewConsole(10, 0),
		editor:        NewEditor(repl.Send),
		qs:            NewQuickSelect(),
		fileBrowser:   NewFileBrowser(),
		sampleBrowser: NewSampleBrowser(),
	}
}

func (a *App) focusEditor() tea.Cmd {
	a.fileBrowser.SetActive(false)
	a.sampleBrowser.SetActive(false)
	if a.active != a.editor {
		a.editor.e.SetMode(vimtea.ModeNormal)
	}
	a.active = a.editor
	return a.editor.e.SetStatusMessage("")
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

func (a *App) openFile(path string) tea.Cmd {
	a.fileBrowser.SetActive(false)
	a.SetActive(a.editor)
	a.editor.e.SetStatusMessage(path)
	return a.editor.load(path)
}

func (a *App) playAudio(path string) tea.Cmd {
	return func() tea.Msg {
		return playAudioMPV(path)
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

	a.fileBrowser.SetDirectory(expandPath(a.cfg.TidalFilesDir))
	a.fileBrowser.SetOnSelect(a.openFile)

	a.sampleBrowser.SetOnSelect(a.playAudio)

	return tea.Batch(
		a.qs.Init(),
		a.fileBrowser.Init(),
		a.sampleBrowser.Init(),
		a.sampleBrowser.SetDirectory(expandPath(a.cfg.SamplesDir)),
		sclangStartCmd(a.sclang),
		replStartCmd(a.repl),
		listenConsole(a.repl.out),
		listenSclang(a.sclang.out),
		a.editor.load("perigee.tidal"),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.qs.SetSize(msg.Width/2, msg.Height/2)
		a.fileBrowser.SetSize(msg.Width/2, msg.Height)
		a.sampleBrowser.SetSize(msg.Width, msg.Height)

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
		case key.Matches(msg, defaultKeyMap.Quit):
			return a, tea.Quit
		case key.Matches(msg, defaultKeyMap.FocusQuickSelect):
			a.qs.SetActive(true)
			a.active = a.qs
			return a, nil
		case key.Matches(msg, defaultKeyMap.FocusFileBrowser):
			a.fileBrowser.SetActive(true)
			a.active = a.fileBrowser
			return a, a.editor.e.SetStatusMessage("file browser focused")
		case key.Matches(msg, defaultKeyMap.FocusAudioBrowser):
			a.sampleBrowser.SetActive(true)
			a.active = a.sampleBrowser
			return a, a.editor.e.SetStatusMessage("audio browser focused")
		case key.Matches(msg, defaultKeyMap.FocusEditor):
			return a, a.focusEditor()
		case key.Matches(msg, defaultKeyMap.FocusConsole):
			a.active = a.activeConsole
			return a, a.editor.e.SetStatusMessage("console focused")
		}
	}

	cmds := []tea.Cmd{}

	if a.activeConsole != nil {
		_, ccmd := a.activeConsole.Update(msg)
		cmds = append(cmds, ccmd)
	}

	_, cmd := a.active.Update(msg)
	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	if a.qs.Active() {
		return a.qs.View()
	}

	if a.fileBrowser.Active() {
		return a.fileBrowser.View()
	}

	if a.sampleBrowser.Active() {
		return a.sampleBrowser.View()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		a.editor.e.View(),
		a.activeConsole.View(),
	)
}
