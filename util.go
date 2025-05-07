package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

var (
	once sync.Once
)

func expandPath(path string) (string) {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return home + path[1:]
}

var audioExts = map[string]struct{}{
	".wav":  {},
	".mp3":  {},
	".flac": {},
	".ogg":  {},
	".aiff": {},
	".aif":  {},
}

func playAudioMPV(path string) error {
	ec := exec.Command("mpv", path)
	return ec.Run()
}

func playAudioBeep(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer f.Close()

	// Use wav decoder instead of mp3
	streamer, format, err := wav.Decode(f)
	if err != nil {
		log.Println("Error decoding WAV:", err)
		return
	}
	defer streamer.Close()

	once.Do(func() {
		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err != nil {
			log.Println("Error initializing speaker:", err)
			return
		}
	})

	log.Println("Playing audio:", filename)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
	})))
}
