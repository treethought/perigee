package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

var circleStyle = lipgloss.NewStyle()

type vizTick time.Time

func harmonicaTick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return vizTick(t)
	})
}

type OscCircle struct {
	X, Y        int
	Size        int
	Color       lipgloss.Color
	Identifier  string
	CreatedAt   time.Time
	Position    float64
	Velocity    float64
	TargetValue float64
}

// HarmonicaVisual implements the Visual interface
type HarmonicaVisual struct {
	width, height int
	active        bool
	circles       map[string]*OscCircle
	lastUpdate    time.Time
	oscMessages   []string
	spring        harmonica.Spring
}

// NewHarmonicaVisual creates a new harmonica-based visual
func NewHarmonicaVisual() *HarmonicaVisual {
	deltaTime := 1.0 / 60.0 // 60 FPS
	angularFrequency := 5.0 // Medium speed
	dampingRatio := 0.2     // Under-damped for some oscillation
	spring := harmonica.NewSpring(deltaTime, angularFrequency, dampingRatio)

	return &HarmonicaVisual{
		circles:     make(map[string]*OscCircle),
		lastUpdate:  time.Now(),
		oscMessages: []string{},
		spring:      spring,
	}
}

// SetSize implements the Visual interface
func (h *HarmonicaVisual) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// SetActive implements the Visual interface
func (h *HarmonicaVisual) SetActive(active bool) {
	h.active = active
}

// Active implements the Visual interface
func (h *HarmonicaVisual) Active() bool {
	return h.active
}

func (h *HarmonicaVisual) Reset() tea.Cmd {
	h.circles = make(map[string]*OscCircle)
	h.oscMessages = []string{}
	h.lastUpdate = time.Now()
	return h.Init()
}

// Init implements the tea.Model interface
func (h *HarmonicaVisual) Init() tea.Cmd {
	return harmonicaTick()
}

// Update implements the tea.Model interface
func (h *HarmonicaVisual) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vizTick:
		// Remove circles older than 1 second
		now := time.Now()
		for id, circle := range h.circles {
			if now.Sub(circle.CreatedAt) > time.Second {
				delete(h.circles, id)
			} else {
				// Update spring animation
				circle.Position, circle.Velocity = h.spring.Update(
					circle.Position,
					circle.Velocity,
					circle.TargetValue,
				)
			}
		}
		h.lastUpdate = now

		return h, harmonicaTick()

	case oscMsg:
		if !h.active {
			return h, harmonicaTick()
		}
		message := string(msg)
		h.handleOscMessage(message)
		return h, harmonicaTick()
	}

	return h, nil
}

// parseOscMessage extracts information from an OSC message
func (h *HarmonicaVisual) parseOscMessage(message string) (string, error) {
	// Example message: "/play ,ssffii kalimba a 0.5 1 1 0"
	if !strings.Contains(message, "/play") {
		return "", fmt.Errorf("not a play message")
	}

	parts := strings.Fields(message)
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid message format")
	}

	// Extract the instrument name (e.g., "kalimba")
	instrument := parts[2]
	return instrument, nil
}

// handleOscMessage processes incoming OSC messages
func (h *HarmonicaVisual) handleOscMessage(message string) {
	instrument, err := h.parseOscMessage(message)
	if err != nil {
		return
	}

	// Log the message for debugging
	h.oscMessages = append(h.oscMessages, message)
	if len(h.oscMessages) > 10 {
		h.oscMessages = h.oscMessages[1:]
	}

	// Ensure we have width and height
	if h.width == 0 {
		h.width = 80
	}
	if h.height == 0 {
		h.height = 24
	}

	// Create a new circle for this instrument
	// Give each instrument a consistent color based on name
	colorIndex := int(instrument[0])
	if len(instrument) > 1 {
		colorIndex += int(instrument[1])
	}

	// Random position but with some margin from edges
	marginX := h.width / 10
	marginY := h.height / 10
	x := marginX + rand.Intn(h.width-2*marginX)
	y := marginY + rand.Intn(h.height-2*marginY)
	log.Println("colorIndex", colorIndex)

	circle := &OscCircle{
		X:           x,
		Y:           y,
		Size:        rand.Intn(3) + 2, // Random size between 2 and 4
		Color:       availableColors()[colorIndex%len(availableColors())],
		Identifier:  instrument,
		CreatedAt:   time.Now(),
		Position:    0.0,
		Velocity:    2.0,
		TargetValue: 1.0,
	}

	// Store the circle using a unique identifier
	uniqueID := fmt.Sprintf("%s_%d", instrument, time.Now().UnixNano())
	h.circles[uniqueID] = circle
}

// availableColors returns a slice of predefined colors
func availableColors() []lipgloss.Color {
	return []lipgloss.Color{
		lipgloss.Color("#FF0000"), // Red
		lipgloss.Color("#00FF00"), // Green
		lipgloss.Color("#0000FF"), // Blue
		lipgloss.Color("#FFFF00"), // Yellow
		lipgloss.Color("#FF00FF"), // Magenta
		lipgloss.Color("#00FFFF"), // Cyan
		lipgloss.Color("#FFA500"), // Orange
		lipgloss.Color("#800080"), // Purple
		lipgloss.Color("#008000"), // Dark Green
		lipgloss.Color("#000080"), // Navy
		lipgloss.Color("#800000"), // Maroon
		lipgloss.Color("#FF69B4"), // Hot Pink
	}
}

// View implements the tea.Model interface
func (h *HarmonicaVisual) View() string {
	// if len(h.circles) == 0 {
	// 	return "✨ Waiting for sound... ✨"
	// }

	// Create a blank canvas
	canvas := make([][]rune, h.height)
	for i := range canvas {
		canvas[i] = make([]rune, h.width)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// Draw each circle on the canvas
	for _, circle := range h.circles {
		drawCircle(canvas, circle)
	}

	// Convert canvas to string
	var sb strings.Builder
	for _, row := range canvas {
		sb.WriteString(string(row) + "\n")
	}

	// Add last few OSC messages as debug info at bottom if there are any
	// if len(h.oscMessages) > 0 {
	// 	sb.WriteString("\nLast message: " + h.oscMessages[len(h.oscMessages)-1])
	// }

	return sb.String()
}

// drawCircle draws a circle on the canvas
func drawCircle(canvas [][]rune, circle *OscCircle) {
	// Get current size based on spring animation
	currentSize := int(float64(circle.Size) * circle.Position)
	if currentSize < 1 {
		currentSize = 1
	}

	// Safety checks
	h := len(canvas)
	if h == 0 {
		return
	}
	w := len(canvas[0])

	x, y := circle.X, circle.Y

	// Draw a circle with the current size as radius
	chars := []rune{'•', '○', '◎', '●', '⬤'}
	char := chars[min(currentSize, len(chars)-1)]

	// For each possible point in the "circle"
	for dy := -currentSize; dy <= currentSize; dy++ {
		for dx := -currentSize; dx <= currentSize; dx++ {
			// Calculate distance from center (simple circle algorithm)
			dist := dx*dx + dy*dy

			// If the point is approximately on the circle's edge or center
			if dist <= currentSize*currentSize {
				nx, ny := x+dx, y+dy
				// Ensure we're within canvas bounds
				if nx >= 0 && nx < w && ny >= 0 && ny < h {
					// Draw more prominent character at the center
					if dx == 0 && dy == 0 {
						canvas[ny][nx] = '●'
					} else if dist <= (currentSize-1)*(currentSize-1) {
						// Fill the inside of larger circles
						canvas[ny][nx] = '·'
					} else {
						canvas[ny][nx] = char
					}
				}
			}
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
