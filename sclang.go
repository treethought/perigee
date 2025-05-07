package main

import (
	"bufio"
	"io"
	"log"
	"os/exec"
)

type SCLangRepl struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	out    chan string
}

func NewSCLangRepl(startupFile string) *SCLangRepl {
	return &SCLangRepl{
		out: make(chan string, 100),
	}
}

func (r *SCLangRepl) buildCmd() *exec.Cmd {
	return exec.Command("pw-jack", "sclang")
}

func (r *SCLangRepl) Start() error {
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

	log.Println("Starting sclang")
	if err := r.cmd.Start(); err != nil {
		return err
	}

	go r.readOutput(r.stdout)
	go r.readOutput(r.stderr)

	return nil
}

func (r *SCLangRepl) Stop() error {
	log.Println("Stopping sclang ...")
	if r.stdin != nil {
		r.stdin.Close()
	}
	if r.cmd.Process != nil {
		return r.cmd.Process.Kill()
	}
	return nil
}

func (r *SCLangRepl) readOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		r.out <- scanner.Text()
	}
}

func (r *SCLangRepl) Output() <-chan string {
	return r.out
}
