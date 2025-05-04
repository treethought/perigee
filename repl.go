package main

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"strings"
)

// TidalRepl starts the tidal process and sends commands to it via stdin and captures its output via stdout.
type TidalRepl struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	out    chan string
}

func NewTidalRepl() *TidalRepl {
	cmd := exec.Command("tidal")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create stderr pipe: %v", err)
	}

	return &TidalRepl{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		out:    make(chan string, 100),
	}
}

func (r *TidalRepl) Start() error {
	log.Println("Starting Tidal REPL...")
	if err := r.cmd.Start(); err != nil {
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
	log.Println("Tidal REPL is not running.")
	log.Println(r.cmd.ProcessState.Pid())
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
	// replace tab with spaces
	cmd = strings.ReplaceAll(cmd, "\t", "  ")
	if _, err := r.stdin.Write([]byte(cmd + "\n")); err != nil {
		return err
	}
	r.out <- cmd
	return nil
}
