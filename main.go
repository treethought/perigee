package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var cfg = &Config{
	Bootfile:      "~/livecoding/tidal/BootTidal.hs",
	TidalFilesDir: "~/livecoding/tidal",
	SamplesDir:    "~/livecoding/tidalsamples",
}

func main() {
	a := NewApp(cfg)

	defer a.repl.Stop()
	defer a.sclang.Stop()

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()
	log.Println("Starting perigee --------------")

	p := tea.NewProgram(a,
		// tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
