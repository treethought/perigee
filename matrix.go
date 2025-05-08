package main

import (
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Character represents a single falling character in the matrix
type Character struct {
	char     string
	row      float64
	col      int
	speed    float64
	modifier float64
	active   bool
	style    lipgloss.Style
}

// MatrixText is a model for rendering text in a matrix-style falling animation
type MatrixText struct {
	text        string
	width       int
	height      int
	viewport    viewport.Model
	characters  []Character
	active      bool
	tick        int
	interval    int // control animation speed
	initialized bool
}

// NewMatrixText creates a new MatrixText model with the given text
func NewMatrixText(text string) *MatrixText {
	v := viewport.New(80, 20)
	v.HighPerformanceRendering = true
	return &MatrixText{
		text:     text,
		viewport: viewport.New(80, 20),
		active:   false,
		interval: 1,
		width:    80,
		height:   20,
	}
}

func (m *MatrixText) SetText(text string) {
	m.text = text
	m.initCharacters()
}

func (m *MatrixText) SetActive(active bool) {
	m.active = active
}

func (m *MatrixText) Active() bool {
	return m.active
}

func (m *MatrixText) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.initCharacters()
}

// Initialize characters from text
func (m *MatrixText) initCharacters() {
	m.characters = []Character{}

	var wsReplace = []string{
		`\`, `/`,
		`\\`, `//`, `|`, `/\`, `\/`, `/`, `/|`, `|/`, `/\/\',\/\/`, `||`,
	}

	// replace all whitespace with a random character

	remainingWhite := strings.Count(m.text, " ")
	for i := rand.Int(); remainingWhite > 0; i += 1 {
		rc := wsReplace[i%len(wsReplace)]
		m.text = strings.Replace(m.text, " ", rc, 1)
		remainingWhite = strings.Count(m.text, " ")
	}

	chars := strings.ReplaceAll(m.text, " ", "")

	// Create a character for each letter in the text
	for i, r := range chars {
		char := string(r)

		// Choose random colors for different effect
		var style lipgloss.Style

		// Create varied styles to make it more dynamic
		colorChoice := rand.Intn(3)
		switch colorChoice {
		case 0:
			// Bright green for primary characters
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
		case 1:
			// Lighter green for secondary characters
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAFFAA"))
		case 2:
			// pink
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
		case 3:
			// blue
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
		default:
			// Dim green for background characters
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#005500"))
		}

		// Distribute across the width
		col := i % m.width
		if col == 0 {
			col = rand.Intn(m.width)
		}
		col = i

		// Randomize starting positions above the viewport
		startRow := float64(-rand.Intn(m.height * 2))

		// Randomize speed for varied motion
		speed := 0.2 + rand.Float64()*0.8

		// Add slight random modifier to prevent uniform motion
		modifier := 0.05 + rand.Float64()*0.1

		m.characters = append(m.characters, Character{
			char:     char,
			row:      startRow,
			col:      col,
			speed:    speed,
			modifier: modifier,
			active:   true,
			style:    style,
		})
	}
}

// Update the matrix animation
func (m *MatrixText) updateMatrix() {
	// Only update every few ticks for performance and to control speed
	m.tick++
	if m.tick < m.interval {
		return
	}
	m.tick = 0

	// Update character positions
	for i := range m.characters {
		if !m.characters[i].active {
			continue
		}

		// Move character down
		m.characters[i].row += m.characters[i].speed

		// if characher has moved pas the bottom, remove it
		if m.characters[i].row > float64(m.height*2) {
			m.characters[i].active = false
		}

		// // If character has moved past the bottom, reset it to the top
		// if m.characters[i].row > float64(m.height*2) {
		// 	m.characters[i].row = float64(-rand.Intn(10))
		// 	// Possibly change column for variety
		// 	if rand.Float64() < 0.3 {
		// 		m.characters[i].col = rand.Intn(m.width)
		// 	}
		// 	// Randomize speed again
		// 	m.characters[i].speed = 0.2 + rand.Float64()*0.8
		// }

		// Occasionally change character speed for more dynamic effect
		if rand.Float64() < 0.01 {
			m.characters[i].speed += (rand.Float64() - 0.5) * m.characters[i].modifier
			if m.characters[i].speed < 0.1 {
				m.characters[i].speed = 0.1
			} else if m.characters[i].speed > 1.5 {
				m.characters[i].speed = 1.5
			}
		}
	}
}

// tickCmd sends a tick message to update the animation
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(time.Time) tea.Msg {
		return matrixTick{}
	})
}

type matrixTick struct{}

func (m *MatrixText) Init() tea.Cmd {
	m.initCharacters()
	return tickCmd()
}

func (m *MatrixText) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	switch msg := msg.(type) {
	case sentMsg:
		m.SetText(string(msg))
		return m, tickCmd()
	case matrixTick:
		m.updateMatrix()
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}

	return m, tickCmd()
}

func (m *MatrixText) View() string {
	if !m.active {
		return ""
	}
	if !m.initialized {
		viewport.Sync(m.viewport)
	}

	// Create a 2D grid to hold characters
	grid := make([][]string, m.height)
	for i := range grid {
		grid[i] = make([]string, m.width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}

	// Place characters on the grid
	for _, c := range m.characters {
		row := int(c.row)
		col := c.col

		// Only show characters that are within the visible area
		if row >= 0 && row < m.height && col >= 0 && col < m.width {
			grid[row][col] = c.style.Render(c.char)
		}
	}

	// Construct the view
	var sb strings.Builder
	for i := 0; i < m.height; i++ {
		for j := 0; j < m.width; j++ {
			sb.WriteString(grid[i][j])
		}
		if i < m.height-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
