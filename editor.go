package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		e: vimtea.NewEditor(
			vimtea.WithFileName("tidal.hs"),
			vimtea.WithStatusStyle(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true),
			),
			// set text background black
			// vimtea.WithTextStyle(lipgloss.NewStyle().Background(lipgloss.Color("#000000"))),
		),
	}

	m.e.AddCommand("q", func(b vimtea.Buffer, a []string) tea.Cmd {
		return func() tea.Msg {
			return tea.Quit()
		}
	})

	m.e.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+_",
		Mode:        vimtea.ModeNormal,
		Description: "Comment line",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			return m.comment(b)
		},
	})
	m.e.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+_",
		Mode:        vimtea.ModeVisual,
		Description: "Comment selected region",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			return m.comment(b)
		},
	})
	m.e.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+e",
		Mode:        vimtea.ModeNormal,
		Description: "Send block to tidal",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			cursor := m.e.GetCursor()

			lines := b.Lines()
			if len(lines) == 0 {
				return vimtea.SetStatusMsg("Buffer is empty")
			}

			// Initialize begin and end to cursor position
			begin := cursor.Row
			end := cursor.Row

			// Find the beginning of the block (go up until empty line or start)
			for i := cursor.Row - 1; i >= 0; i-- {
				if strings.TrimSpace(lines[i]) == "" {
					break
				}
				begin = i
			}

			// Find the end of the block (go down until empty line or end)
			for i := cursor.Row + 1; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) == "" {
					break
				}
				end = i
			}

			// Extract lines for the block (inclusive)
			blockLines := lines[begin : end+1]
			content := strings.Join(blockLines, "\n")

			if err := m.send(content); err != nil {
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
			return m.save("perigee.tidal", b.Text())
		},
	})

	return m
}

func (m *Editor) comment(b vimtea.Buffer) tea.Cmd {
	cursor := m.e.GetCursor()
	lines := b.Lines()

	start, end := cursor, cursor
	if m.e.GetMode() == vimtea.ModeVisual {
		start, end = m.e.GetSelectionBoundary()
		log.Printf("visual start: %v, end: %v", start, end)
	}

	for r := start.Row; r <= end.Row; r++ {
		if len(lines) == 0 || r >= len(lines) {
			continue
		}

		line := lines[r]
		trimmedLine := strings.TrimLeft(line, " \t")
		indentSize := len(line) - len(trimmedLine)

		commentPrefix := "-- "
		if strings.HasPrefix(trimmedLine, commentPrefix) {
			b.DeleteAt(r, indentSize, r, indentSize+2)
			continue
		}
		b.InsertAt(r, indentSize, commentPrefix)
	}
	return nil
}

func (m *Editor) load(fname string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(fname)
		if err != nil {
			log.Printf("Error loading file %s: %v", fname, err)
			return m.e.SetStatusMessage(fmt.Sprintf("Error loading file: %v", err))
		}
		m.e.GetBuffer().InsertAt(0, 0, string(content))
		return m.e.SetStatusMessage(fmt.Sprintf("File %s loaded!", fname))
	}
}

func (m *Editor) save(fname string, content string) tea.Cmd {
	return func() tea.Msg {
		err := os.WriteFile(fname, []byte(content), 0644)
		if err != nil {
			log.Printf("Error saving file %s: %v", fname, err)
			return m.e.SetStatusMessage(fmt.Sprintf("Error saving file: %v", err))
		}
		log.Printf("File %s saved successfully", fname)
		return m.e.SetStatusMessage(fmt.Sprintf("File %s saved!", fname))
	}
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
	log.Print("editor UPDATE")
	log.Print(msg)
	model, cmd := m.e.Update(msg)
	if model != nil {
		m.e = model.(vimtea.Editor)
	}
	return m, cmd
}

func (m *Editor) View() string {
	return m.e.View()
}
