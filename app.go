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
	Quit                key.Binding
	FocusEditor         key.Binding
	FocusConsole        key.Binding
	ToggleReplConsole   key.Binding
	ToggleSclangConsole key.Binding
	FocusQuickSelect    key.Binding
	FocusFileBrowser    key.Binding
	ToggleAudioBrowser  key.Binding
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
		key.WithHelp("2", "toggle console focus"),
	),
	ToggleReplConsole: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "toggle repl console"),
	),
	ToggleSclangConsole: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "toggle console"),
	),

	FocusQuickSelect: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open quick select"),
	),
	FocusFileBrowser: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "open file browser"),
	),
	ToggleAudioBrowser: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("ctrl+w", "toggle audio browser"),
	),
}

type App struct {
	cfg           *Config
	editor        *Editor
	repl          *TidalRepl
	replConsole   *Console
	sclang        *SCLangRepl
	scConsole     *Console
	qs            *QuickSelect
	fileBrowser   *FileBrowser
	sampleBrowser *SampleBrowser
	active        tea.Model
	activeConsole *Console
	h, w          int
}

func NewApp(cfg *Config) *App {
	repl := NewTidalRepl(cfg.Bootfile)
	sclang := NewSCLangRepl("")
	return &App{
		cfg:           cfg,
		repl:          repl,
		sclang:        sclang,
		replConsole:   NewConsole(0, 10),
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
		a.activeConsole = a.replConsole
	default:
		a.activeConsole = a.replConsole
	}
}

func (a *App) openFile(path string) tea.Cmd {
	a.fileBrowser.SetActive(false)
	a.SetActive(a.editor)
	return tea.Sequence(
		a.editor.load(path),
		a.editor.e.SetStatusMessage(path),
	)
}

func (a *App) playAudio(path string) tea.Cmd {
	return func() tea.Msg {
		return playAudioMPV(path)
	}
}

func (a *App) Init() tea.Cmd {
	a.activeConsole = a.replConsole
	a.activeConsole.SetActive(true)
	a.SetActive(a.editor)
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
		a.editor.Init(),
		a.editor.load("perigee.tidal"),
	)
}

func (a *App) SetSize(width, height int) {
	a.w = width
	a.h = height
	// pop over so get desired height without calcs
	a.qs.SetSize(a.w/2, a.h/2)
	a.fileBrowser.SetSize(a.w/2, a.h)

	ch, sw := 0, 0

	// console height
	if a.activeConsole != nil {
		ch = a.h / 4
		if ch < 10 {
			ch = 10
		}
		a.replConsole.SetSize(a.w, ch)
		a.scConsole.SetSize(a.w, ch)
	}
	// side samples width
	if a.sampleBrowser.Active() {
		sw = a.w / 3
		if sw < 16 {
			sw = 16
		}
		a.sampleBrowser.SetSize(sw, a.h-ch-3)
	}

	a.editor.SetSize(a.w-sw, a.h-ch-1) // Reserve space for console
	return
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.SetSize(msg.Width, msg.Height)

	case consoleMsg:
		a.replConsole.AddLine(string(msg))
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
		case key.Matches(msg, defaultKeyMap.FocusConsole):
			a.active = a.activeConsole
			return a, a.editor.e.SetStatusMessage("console")
		case key.Matches(msg, defaultKeyMap.ToggleReplConsole):
			a.scConsole.SetActive(a.replConsole.Active())
			a.replConsole.SetActive(!a.replConsole.Active())
			if a.replConsole.Active() {
				a.activeConsole = a.replConsole
				a.active = a.activeConsole
				a.SetSize(a.w, a.h)
				return a, a.editor.e.SetStatusMessage("repl console")
			}
			a.activeConsole = nil
			a.SetSize(a.w, a.h)
			return a, a.focusEditor()
		case key.Matches(msg, defaultKeyMap.ToggleSclangConsole):
			a.replConsole.SetActive(a.scConsole.Active())
			a.scConsole.SetActive(!a.scConsole.Active())
			if a.scConsole.Active() {
				a.activeConsole = a.scConsole
				a.active = a.activeConsole
				a.SetSize(a.w, a.h)
				return a, a.editor.e.SetStatusMessage("sclang console")
			}
			a.activeConsole = nil
			a.SetSize(a.w, a.h)
			return a, a.focusEditor()
		case key.Matches(msg, defaultKeyMap.ToggleAudioBrowser):
			a.sampleBrowser.SetActive(!a.sampleBrowser.Active())
			if a.sampleBrowser.Active() {
				a.active = a.sampleBrowser
				a.SetSize(a.w, a.h)
				return a, a.editor.e.SetStatusMessage("audio browser")
			}
			a.active = a.editor
			a.SetSize(a.w, a.h)
			return a, nil
		case key.Matches(msg, defaultKeyMap.FocusEditor):
			a.SetSize(a.w, a.h)
			return a, a.focusEditor()
		}
	}

	cmds := []tea.Cmd{}

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
	cv := ""
	if a.activeConsole != nil && a.activeConsole.Active() {
		cv = a.activeConsole.View()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			a.editor.View(),
			a.sampleBrowser.View(),
		),
		cv,
	)
}
