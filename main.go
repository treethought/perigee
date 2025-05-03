package main

import (
	"fmt"
	"log"
	"os"

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

type App struct {
	editor  vimtea.Editor
	repl    *TidalRepl
	console *Console
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		replStartCmd(a.repl),
		listenConsole(a.repl.out),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.console.SetSize(msg.Width, 10)
		ed, cmd := a.editor.SetSize(msg.Width, msg.Height-10) // Reserve space for console
		a.editor = ed.(vimtea.Editor)
		return a, tea.Batch(
			cmd,
		)

	case consoleMsg:
		log.Printf("Console message: %s", msg)
		a.console.AddLine(string(msg))
		return a, listenConsole(a.repl.out)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
	}
	editor, cmd := a.editor.Update(msg)
	a.editor = editor.(vimtea.Editor)
	return a, cmd
}

func (a *App) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		a.editor.View(),
		a.console.View(),
	)
}

func main() {
	a := &App{
		editor:  vimtea.NewEditor(),
		repl:    NewTidalRepl(),
		console: NewConsole(0, 10), // Adjust width and height as needed
	}

	a.editor.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+e",
		Mode:        vimtea.ModeNormal,
		Description: "Send buffer to tidal",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			if err := a.repl.Send(b.Text()); err != nil {
				return vimtea.SetStatusMsg(fmt.Sprintf("Error sending command: %v", err))
			}
			return vimtea.SetStatusMsg("sent!")
		},
	})

	a.editor.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+h",
		Mode:        vimtea.ModeNormal,
		Description: "Hush",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			if err := a.repl.Send("hush"); err != nil {
				return vimtea.SetStatusMsg(fmt.Sprintf("Error sending command: %v", err))
			}
			return vimtea.SetStatusMsg("Hushed!")
		},
	})

	a.editor.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+s",
		Mode:        vimtea.ModeNormal,
		Description: "Save file",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			return vimtea.SetStatusMsg("File saved!")
		},
	})

	defer a.repl.Stop()

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(a)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
