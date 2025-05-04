package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var bootfile = ""

func main() {
	repl := NewTidalRepl(bootfile)
	sclang := NewSCLangRepl("")
	a := NewApp(repl, sclang)

	defer a.repl.Stop()
	defer a.sclang.Stop()

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(a,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
