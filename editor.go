package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/vimtea"
)

var defaultFile = "perigee.tidal"

type sendFunc func(s string) error

type sentMsg string

func sentMsgCmd(s string) tea.Cmd {
	return func() tea.Msg {
		return sentMsg(s)
	}
}

// example osc msg
// we need to map patterns we send (blocks) to tidal to editor indexes
// then when receiving the osc msg like below, we parse it and highlight the sound (and later more pieces) in the editor
// /play ,sssfsfsfsfsfsfsiss _id_ 3 begin 0 cps 0.5625 cycle 116 delta 3.555552 end 0.5 n 7 orbit 2 s vspectral192

type Pos struct {
	Row, Col int
}

// store indexes of pattern components
type PatternData struct {
	// i.e d1/d2 __id in osc msg
	ID     string
	sounds map[string]Pos
}

type Editor struct {
	e           vimtea.Editor
	send        sendFunc
	currentFile string
	prevFile    string

	// map of pattern id to pattern data
	patterns map[string]PatternData
}

func NewEditor(send sendFunc) *Editor {
	m := &Editor{
		currentFile: defaultFile,
		send:        send,
		patterns:    make(map[string]PatternData),
		e: vimtea.NewEditor(
			vimtea.WithFileName("tidal.hs"),
			vimtea.WithDefaultSyntaxTheme("autumn"),
			vimtea.WithStatusStyle(
				lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true),
			),
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
		Key:         "ctrl+^",
		Mode:        vimtea.ModeNormal,
		Description: "Previous file",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			if m.prevFile == "" {
				return vimtea.SetStatusMsg("No previous file")
			}
			return m.load(m.prevFile)
		},
	})
	m.e.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+e",
		Mode:        vimtea.ModeNormal,
		Description: "Send block to tidal",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			blockLines, _, _ := m.getCursorBlock(b)
			if len(blockLines) == 0 {
				return vimtea.SetStatusMsg("No block selected")
			}
			m.storePatternData(blockLines, 0, len(blockLines)-1)
			content := strings.Join(blockLines, "\n")

			if err := m.send(content); err != nil {
				return vimtea.SetStatusMsg(fmt.Sprintf("Error sending command: %v", err))
			}
			return tea.Batch(
				vimtea.SetStatusMsg("sent!"),
				sentMsgCmd(content),
			)
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
			return m.save(m.currentFile, b.Text())
		},
	})

	return m
}

// patternId matches d1 d2 s1 s2, or any non-quoted string that is a letter followed by a number
// and is the first non-whitespace character on the line

// sound matches any string following a space after an "s" after # or $
// we also need to match every sound in a pattern
// should not match s in other keeywods like $ slow 2
// should also match sound
// # s "kalimba"      --> [kalimba]
// # s "uku gtr piano" --> [uku, gtr, piano]
// # s "[gtr, cpu]" --> [gtr cpu]
// # s "<vibe {piano, steeldrum}%4" -> [vibe, piano, steeldrum]
// above result after -> is the list of matches. we want each sound present given without any surrouunding quotes/brackets/commas etc to be given in the slice of matches

var (
	patternIdRegex = regexp.MustCompile(`(?m)^\s*([a-zA-Z]\d*)\s*([^\s]+)`)
	soundRegex     = regexp.MustCompile(`(?m)([#\$])\s+(s|sound)\s+(.+?)(?:\s*#|$)`)
)

func extractSoundsFromLineRegex(line string) []string {
	// Match the sound pattern
	soundMatches := soundRegex.FindAllStringSubmatch(line, -1)
	var sounds []string
	for _, match := range soundMatches {
		// Extract the sound value from the match (3rd capture group)
		soundValue := match[3]
		// Remove outer quotes and brackets
		soundValue = strings.Trim(soundValue, "\"[] ")
		
		// Split by comma to handle multiple sounds like "[bsperc(<4 6>, 4), superhat(4, 8)]"
		soundParts := strings.Split(soundValue, ",")
		for _, part := range soundParts {
			part = strings.TrimSpace(part)
			// Extract just the sound name before any parentheses or special chars
			if idx := strings.IndexAny(part, "(<{"); idx != -1 {
				part = part[:idx]
			}
			part = strings.TrimSpace(part)
			if part != "" {
				sounds = append(sounds, part)
			}
		}
	}
	return sounds
}

type idBlock struct {
	id         string
	lines      []string
	start, end int
}

// oscSoundRegex is a simple match for the string after s
// always <space>s<space> and then the sound name
var oscSoundRegex = regexp.MustCompile(`(?m)^\s*s\s*["<\[]?([^\s,]+)([">,\]]?)`)

func (m *Editor) parseOscMsg(msg string) (id string, sounds []string) {
	// Parse the OSC message to extract pattern data
	// This is a placeholder implementation
	// You need to implement the actual parsing logic based on your OSC message format
	log.Printf("parsing OSC message: %s", msg)

	// osc msgs look like this. for now we are only interested in the pattern id and sounds
	//  /play ,sssfsfsfsfsiss _id_ 1 cps 0.5625 cycle 38.6 delta 0.177777 note 3 orbit 0 s kalimba            │
	//  /play ,sssfsfsfsfsiss _id_ 1 cps 0.5625 cycle 38.7 delta 0.355555 note 7 orbit 0 s kalimba            │
	//  /play ,sssfsfsfsfsiss _id_ 1 cps 0.5625 cycle 38.9 delta 0.059259 note 12 orbit 0 s kalimba           │
	//  /play ,sssfsfsfsfsiss _id_ 1 cps 0.5625 cycle 38.933334 delta 0.05926 note 12 orbit 0 s kalimba       │
	//  /play ,sssfsfsfsfsiss _id_ 1 cps 0.5625 cycle 38.966667 delta 0.059259 note 12 orbit 0 s kalimba      │
	//  /play ,sssfsfsfsfsiss _id_ 1 cps 0.5625 cycle 39 delta 0.444444 note 0 orbit 0 s kalimba              │

	if strings.Contains(msg, "_id_") {
		parts := strings.Split(msg, "_id_")
		if len(parts) > 1 {
			id = strings.TrimSpace(parts[1])
			id = strings.Split(id, " ")[0]
			log.Printf("found pattern id: %s", id)
		}
		soundParts := strings.Split(msg, " ")
		if idx := slices.Index(soundParts, "s"); idx > -1 {
			if len(soundParts) > idx+1 {
				sound := soundParts[idx+1]
				log.Printf("found sound: %s", sound)
				return id, []string{sound}
			}
		}

		// Extract sounds from the message. osc msgs are simpler than the block lines
		// there are no brackes or anything, always a single sound after s
		soundMatches := oscSoundRegex.FindAllStringSubmatch(msg, -1)
		for _, match := range soundMatches {
			sound := match[1]
			sound = strings.Trim(sound, "\"[] ,")
			sounds = append(sounds, sound)
		}
	}
	return id, sounds
}

func (m *Editor) highlightSound(b vimtea.Buffer, id, sound string) {
	log.Printf("highlighting sound %s in pattern %s", sound, id)
	patternData, ok := m.patterns[id]
	if !ok {
		log.Printf("Pattern ID %s not found", id)
		for i, _ := range m.patterns {
			log.Printf("Pattern ID %s", m.patterns[i].ID)
		}

		return
	}
	// Check if the sound exists in the pattern PatternData
	pos, ok := patternData.sounds[sound]
	if !ok {
		log.Printf("Sound %s not found in pattern ID %s", sound, id)
		return
	}
	log.Printf("found sound %s at %d:%d", sound, pos.Row, pos.Col)

	m.e.HighlightText(pos.Row, pos.Col, pos.Row, pos.Col+len(sound), time.Millisecond*200)

	// b.DeleteAt(pos.Row, pos.Col, pos.Row, pos.Col+len(sound))
	// b.InsertAt(pos.Row, pos.Col, lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render(sound))
	//
}

func (m *Editor) storePatternData(lines []string, startRow, endRow int) {
	// Parse the lines to extract pattern storePatternData

	var idBlocks []idBlock

	// first separate all pattern id blocks, i.e split d1 d2 etc
	// d1 /d2 etc must be the start of a line
	for i := startRow; i <= endRow; i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		// check if line starts with a pattern _id_
		// then continue through until we fine blank line or next pattern id
		if patternIdRegex.MatchString(line) {
			matches := patternIdRegex.FindStringSubmatch(line)
			log.Println("PATTERN MATCHEs", matches)
			id := matches[2]
			idBlock := idBlock{id: id, lines: lines, start: i}

			// now we need to find the end of this blocks
			for j := i + 1; j <= endRow; j++ {
				nextLine := lines[j]
				if strings.TrimSpace(nextLine) == "" {
					idBlock.end = j - 1
					break
				}
				if patternIdRegex.MatchString(nextLine) {
					idBlock.end = j - 1
					break
				}
			}
			if idBlock.end == 0 {
				idBlock.end = endRow
			}
			idBlocks = append(idBlocks, idBlock)
		}
	}
	// now we have all the pattern ids, we need to extract all sounds from each pattern's lines

	for _, block := range idBlocks {
		id := block.id
		start := block.start
		end := block.end
		lines := block.lines
		for i := start; i <= end; i++ {
			if i >= len(lines) {
				log.Printf("Index %d out of range for lines", i)
			}
			line := lines[i]
			if strings.TrimSpace(line) == "" {
				continue
			}
			data, ok := m.patterns[id]
			if !ok {
				data = PatternData{
					ID:     id,
					sounds: make(map[string]Pos),
				}
			}
			// Extract sounds from the line
			log.Println("check line for sounds: ", line)
			sounds := extractSoundsFromLineRegex(line)
			for _, sound := range sounds {
				// Store the sound and its position (convert to absolute buffer position)
				data.sounds[sound] = Pos{Row: startRow + i, Col: strings.Index(line, sound)}
			}
			m.patterns[id] = data

		}
	}

	for id, data := range m.patterns {
		log.Printf("Pattern ID: %s, Sounds: %v", id, data.sounds)
	}

}

func (m *Editor) getCursorBlock(b vimtea.Buffer) (rows []string, startRow, endRow int) {
	cursor := m.e.GetCursor()

	lines := b.Lines()
	if len(lines) == 0 {
		return nil, 0, 0
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
	return blockLines, begin, end
}

func (m *Editor) comment(b vimtea.Buffer) tea.Cmd {
	cursor := m.e.GetCursor()
	lines := b.Lines()

	start, end := cursor, cursor
	if m.e.GetMode() == vimtea.ModeVisual {
		start, end = m.e.GetSelectionBoundary()
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
		if m.currentFile != "" {
			m.prevFile = m.currentFile
		}
		content, err := os.ReadFile(fname)
		if err != nil {
			log.Printf("Error loading file %s: %v", fname, err)
			return m.e.SetStatusMessage(fmt.Sprintf("Error loading file: %v", err))
		}
		m.e.GetBuffer().Clear()
		m.e.GetBuffer().InsertAt(0, 0, string(content))
		m.currentFile = fname
		return m.e.SetStatusMessage(fname)
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
		return m.e.SetStatusMessage(fmt.Sprintf("saved: %s", fname))
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
	switch msg := msg.(type) {
	case oscMsg:
		id, sounds := m.parseOscMsg(string(msg))
		if id != "" {
			m.highlightSound(m.e.GetBuffer(), id, sounds[0])
		}
		log.Printf("parsed osc msg: , id: %s, sounds: %v", id, sounds)
	}
	_, cmd := m.e.Update(msg)
	// if model != nil {
	// 	m.e = model.(vimtea.Editor)
	// }
	return m, cmd
}

func (m *Editor) View() string {
	return m.e.View()
}
