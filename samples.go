package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SampleBrowser wraps an audiobrowser and builds all samples from directories of tidal samples
type SampleBrowser struct {
	active   bool
	rootDir  string
	onSelect func(path string) tea.Cmd
	samples  map[string][]audioFile
	ab       *AudioBrowser
}

func NewSampleBrowser() *SampleBrowser {
	// Start in the current directory
	curDir, err := os.Getwd()
	if err != nil {
		curDir = "."
	}
	m := &SampleBrowser{
		ab:      NewAudioBrowser(),
		rootDir: curDir,
	}

	return m
}

func (m *SampleBrowser) SetSize(width, height int) {
	m.ab.SetSize(width, height)
}

func (m *SampleBrowser) SetActive(active bool) {
	m.ab.SetActive(active)
	m.active = active
}

func (m *SampleBrowser) Active() bool {
	return m.active
}

func (m *SampleBrowser) SetOnSelect(f func(path string) tea.Cmd) {
	m.ab.SetOnSelect(f)
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size < KB:
		return fmt.Sprintf("%d B", size)
	case size < MB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	case size < GB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	default:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	}
}

// LoadSamples recursively finds all audio samples in a directory
// and organizes them by their parent folder name
func loadSampleMap(rootDir string) (map[string][]audioFile, error) {
	log.Println("Loading samples from directory:", rootDir)
	samples := make(map[string][]audioFile)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include audio files
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := audioExts[ext]; !ok {
			return nil
		}

		// Get parent directory name as the sample bank name
		parentDir := filepath.Base(filepath.Dir(path))

		// Create sample entry
		sample := audioFile{
			path:     path,
			name:     filepath.Base(path),
			fileType: getFileType(path),
			size:     formatSize(info.Size()),
		}

		// Add to map
		samples[parentDir] = append(samples[parentDir], sample)

		return nil
	})

	return samples, err
}

func (m *SampleBrowser) loadSamples() tea.Cmd {
	return func() tea.Msg {
		log.Println("------loading samples------")
		samples, err := loadSampleMap(m.rootDir)
		if err != nil {
			log.Println("Error loading samples:", err)
			return nil
		}
		m.samples = samples
		for _, files := range samples {
			sort.Slice(files, func(i, j int) bool {
				return strings.ToLower(files[i].name) < strings.ToLower(files[j].name)
			})
		}

		log.Println("Adding:", len(samples), "banks to audiobrowser")
		return m.ab.SetFiles(samples)
	}

}

func (m *SampleBrowser) SetDirectory(path string) tea.Cmd {
	m.rootDir = path
	return m.loadSamples()
}

func (m *SampleBrowser) Init() tea.Cmd {
	return nil
}

func (m *SampleBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.ab.Update(msg)
	return m, cmd
}

func (m *SampleBrowser) View() string {
	if !m.active {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder())

	return style.Render(m.ab.View())
}

func getFileType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".wav":
		return "WAV"
	case ".mp3":
		return "MP3"
	case ".flac":
		return "FLAC"
	case ".ogg":
		return "OGG"
	case ".aiff", ".aif":
		return "AIFF"
	case ".alac":
		return "ALAC"
	case ".midi", ".mid":
		return "MIDI"
	case "":
		return "DIR" // For directories
	default:
		return ext
	}
}
