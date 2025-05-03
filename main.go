package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	editor  *Editor
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
		_, cmd := a.editor.SetSize(msg.Width, msg.Height-10) // Reserve space for console
		return a, cmd

	case consoleMsg:
		log.Printf("Console message: %s", msg)
		a.console.AddLine(string(msg))
		return a, listenConsole(a.repl.out)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
	}
	_, cmd := a.editor.Update(msg)
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
	repl := NewTidalRepl()
	a := &App{
		repl:    repl,
		console: NewConsole(0, 10),
		editor:  NewEditor(repl.Send),
	}

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
