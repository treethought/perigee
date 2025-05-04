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
		bootFile: bootfile,
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
