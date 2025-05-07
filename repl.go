package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TidalRepl starts the tidal process and sends commands to it via stdin and captures its output via stdout.
type TidalRepl struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	out      chan string
	bootFile string // Path to the boot file, if any
}

func NewTidalRepl(bootFile string) *TidalRepl {
	return &TidalRepl{
		out:      make(chan string, 100),
		bootFile: expandPath(bootFile),
	}
}

func (r *TidalRepl) buildCmd() *exec.Cmd {
	if r.bootFile != "" {
		cmd := exec.Command("ghci", "-ghci-script", r.bootFile)
		cmd.Dir = filepath.Dir(r.bootFile)
		return cmd
	}
	log.Println("no BootTidal.hs, launching without")
	return exec.Command("tidal")
}

func (r *TidalRepl) Start() error {
	if r.bootFile == "" {
		r.bootFile, _ = findFileUpwards("BootTidal.hs")
	}

	r.cmd = r.buildCmd()
	var err error

	r.stdin, err = r.cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe: %v", err)
	}

	r.stdout, err = r.cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}
	r.stderr, err = r.cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create stderr pipe: %v", err)
	}

  log.Printf("Starting Tidal REPL: %s %s", r.cmd.Path, r.cmd.Args)
	if err := r.cmd.Start(); err != nil {
		log.Printf("Failed to start Tidal REPL: %v", err)
		return err
	}

	go r.readOutput(r.stdout)
	go r.readOutput(r.stderr)
	return nil
}

func (r *TidalRepl) Stop() error {
	log.Println("Stopping Tidal REPL...")
	if r.stdin != nil {
		r.stdin.Close()
	}

	if r.cmd.Process != nil {
		return r.cmd.Process.Kill()
	}
	return nil
}

func (r *TidalRepl) readOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		r.out <- scanner.Text()
	}
}

func (r *TidalRepl) Output() <-chan string {
	return r.out
}

func (r *TidalRepl) Send(cmd string) error {
	escaped := r.escapeText(cmd)
	if _, err := r.stdin.Write([]byte(escaped)); err != nil {
		return err
	}
	r.out <- cmd
	return nil
}

// escapeTextTidal mimics the vim-tidal _EscapeText_tidal function
func (r *TidalRepl) escapeText(text string) string {
	// tabs aren't allowed
	text = strings.ReplaceAll(text, "\t", "  ")
	lines := strings.Split(text, "\n")

	lines = r.wrapIfMulti(lines)
	return strings.Join(lines, "\n") + "\n"
}

// wrapIfMulti wraps lines in :{ :} if there's more than one line
func (r *TidalRepl) wrapIfMulti(lines []string) []string {
	if len(lines) > 1 {
		// Prepend :{ and append :}
		wrapped := make([]string, 0, len(lines)+2)
		wrapped = append(wrapped, ":{")
		wrapped = append(wrapped, lines...)
		wrapped = append(wrapped, ":}")
		return wrapped
	}
	return lines
}

func findFileUpwards(filename string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(dir, filename)

		// Check if file exists
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		// Get parent directory
		parent := filepath.Dir(dir)

		// If we've reached the root, stop
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", fmt.Errorf("file %s not found", filename)
}
