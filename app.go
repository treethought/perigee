package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/vimtea"
	posc "github.com/treethought/perigee/osc"
)

type tidalMsg string
type sclangMsg string
type oscMsg string

func listenSclang(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return sclangMsg(<-ch)
	}
}

func listenTidal(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return tidalMsg(<-ch)
	}
}

func listenOsc(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return oscMsg(<-ch)
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
func oscStartCmd(osc *posc.Server) tea.Cmd {
	return func() tea.Msg {
		if err := osc.Start(); err != nil {
			return oscMsg(err.Error())
		}
		return nil
	}
}

type keyMap struct {
	Quit                key.Binding
	FocusEditor         key.Binding
	FocusConsole        key.Binding
	ToggleTidalConsole  key.Binding
	ToggleSclangConsole key.Binding
	ToggleOscConsole    key.Binding
	FocusQuickSelect    key.Binding
	FocusFileBrowser    key.Binding
	ToggleAudioBrowser  key.Binding
	ToggleVisuals       key.Binding
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
	ToggleTidalConsole: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "toggle tidal console"),
	),
	ToggleSclangConsole: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "toggle console"),
	),
	ToggleOscConsole: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "toggle osc console"),
	),
	// FocusQuickSelect: key.NewBinding(
	// 	key.WithKeys("ctrl+o"),
	// 	key.WithHelp("ctrl+o", "open quick select"),
	// ),
	FocusFileBrowser: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "open file browser"),
	),
	ToggleAudioBrowser: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("ctrl+w", "toggle audio browser"),
	),
	ToggleVisuals: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "toggle visuals"),
	),
}

type App struct {
	cfg    *Config
	editor *Editor
	osc    *posc.Server
	// oscConsole    *Console
	repl *TidalRepl
	// replConsole   *Console
	sclang *SCLangRepl
	// scConsole     *Console
	qs            *QuickSelect
	fileBrowser   *FileBrowser
	sampleBrowser *SampleBrowser
	visuals       *VisualsView
	active        tea.Model
	activeConsole *Console
	h, w          int
	consoles      map[string]*Console
}

func NewApp(cfg *Config) *App {
	osc := posc.NewServer(9191)
	repl := NewTidalRepl(cfg.Bootfile)
	sclang := NewSCLangRepl("")
	matrix := NewMatrixText("perigee")
	harmonicaVisual := NewHarmonicaVisual()
	visuals := NewVisualsView(map[string]Visual{
		"matrix":    matrix,
		"harmonica": harmonicaVisual,
	})

	consoles := map[string]*Console{
		"osc":    NewConsole(0, 0),
		"sclang": NewConsole(0, 0),
		"tidal":  NewConsole(0, 0),
	}

	return &App{
		cfg:           cfg,
		osc:           osc,
		repl:          repl,
		sclang:        sclang,
		consoles:      consoles,
		editor:        NewEditor(repl.Send),
		qs:            NewQuickSelect(),
		fileBrowser:   NewFileBrowser(),
		sampleBrowser: NewSampleBrowser(),
		visuals:       visuals,
	}
}

func (a *App) focusEditor() tea.Cmd {
	a.fileBrowser.SetActive(false)
	// a.sampleBrowser.SetActive(false)
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
	if c, ok := a.consoles[val]; ok {
		a.activeConsole = c
		return
	}
	a.activeConsole = a.consoles["tidal"]
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
	a.visuals.SetActiveModel("harmonica")
	a.setActiveConsole("tidal")
	a.activeConsole.SetActive(true)

	a.SetActive(a.editor)
	a.visuals.SetActive(false)
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
		a.editor.Init(),
		a.qs.Init(),
		a.fileBrowser.Init(),
		a.sampleBrowser.Init(),
		a.visuals.Init(),
		a.sampleBrowser.SetDirectory(expandPath(a.cfg.SamplesDir)),
		sclangStartCmd(a.sclang),
		replStartCmd(a.repl),
		oscStartCmd(a.osc),
		listenTidal(a.repl.out),
		listenSclang(a.sclang.out),
		listenOsc(a.osc.Out()),
		a.editor.load("perigee.tidal"),
	)
}

func (a *App) SetSize(width, height int) {
	a.w = width
	a.h = height
	// pop over so get desired height without calcs
	a.qs.SetSize(a.w/2, a.h/2)
	a.fileBrowser.SetSize(a.w/2, a.h)

	ch, vw, sw := 0, 0, 0

	// console height
	if a.activeConsole != nil {
		ch = a.h / 4
		if ch < 10 {
			ch = 10
		}
		for _, c := range a.consoles {
			c.SetSize(a.w, ch)
		}
	}

	if a.visuals.Active() {
		vw = a.w / 3
		if vw < 10 {
			vw = 10
		}
		a.visuals.SetSize(vw, a.h-ch-3)
	}

	// side samples width
	if a.sampleBrowser.Active() {
		sw = a.w / 3
		if sw < 16 {
			sw = 16
		}
		a.sampleBrowser.SetSize(sw, a.h-ch-3)
	}

	a.editor.SetSize(a.w-sw-vw, a.h-ch-1) // Reserve space for console
	return
}

func (a *App) selectConsole(c string) tea.Cmd {
	var selected *Console

	for n, cc := range a.consoles {
		if n == c {
			cc.SetActive(true)
			selected = cc
			continue
		}
		cc.SetActive(false)
	}

	if selected == nil || selected == a.activeConsole {
		a.activeConsole.SetActive(false)
		a.activeConsole = nil
		a.SetSize(a.w, a.h)
		return a.focusEditor()
	}

	a.activeConsole = selected
	a.active = a.activeConsole
	a.SetSize(a.w, a.h)

	if a.activeConsole != nil {
		return a.editor.e.SetStatusMessage(c)
	}
	return a.focusEditor()

}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.SetSize(msg.Width, msg.Height)
		return a, nil

	case tidalMsg:
		a.consoles["tidal"].AddLine(string(msg))
		return a, listenTidal(a.repl.out)

	case sclangMsg:
		a.consoles["sclang"].AddLine(string(msg))
		return a, listenSclang(a.sclang.out)

	case oscMsg:
		cmds = append(cmds, listenOsc(a.osc.Out()))
		if a.consoles["osc"].Active() {
			a.consoles["osc"].AddLine(string(msg))
		}
		// Pass OSC message to the visuals model first
		if a.visuals.Active() && a.visuals.activeModel != nil {
			_, vcmd := a.visuals.activeModel.Update(msg)
			cmds = append(cmds, vcmd)
		}
		return a, tea.Batch(cmds...)

	case tea.KeyMsg:

		if a.active == nil {
			a.active = a.editor
		}

		if a.active == a.editor && (a.editor.e.GetMode() != vimtea.ModeNormal) {
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
		case key.Matches(msg, defaultKeyMap.ToggleTidalConsole):
			return a, a.selectConsole("tidal")
		case key.Matches(msg, defaultKeyMap.ToggleSclangConsole):
			return a, a.selectConsole("sclang")
		case key.Matches(msg, defaultKeyMap.ToggleOscConsole):
			cmds = []tea.Cmd{a.selectConsole("osc")}
			if !a.consoles["osc"].Active() {
				cmds = append(cmds, listenOsc(a.osc.Out()))
			}
			return a, tea.Batch(cmds...)

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
		case key.Matches(msg, defaultKeyMap.ToggleVisuals):
			// TODO: determine if we need to listen for osc based on active visual
			// currently enabling if osc console hasn't been activated to start osc listen
			var cmd tea.Cmd = nil
			if !a.consoles["osc"].Active() {
				cmd = listenOsc(a.osc.Out())
			}
			a.visuals.SetActive(!a.visuals.Active())
			if a.visuals.Active() {
				// no need to focus visuals
				a.SetSize(a.w, a.h)
				return a, tea.Batch(cmd, a.editor.e.SetStatusMessage("visuals enabled"))
			}
			a.active = a.editor
			a.SetSize(a.w, a.h)
			return a, cmd
		case key.Matches(msg, defaultKeyMap.FocusEditor):
			a.SetSize(a.w, a.h)
			return a, a.focusEditor()
		}
	}

	if a.visuals.Active() {
		_, vmsg := a.visuals.Update(msg)
		cmds = append(cmds, vmsg)
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

	vv := ""
	if a.visuals.Active() {
		vv = a.visuals.View()
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
			vv,
			a.sampleBrowser.View(),
		),
		cv,
	)
}
