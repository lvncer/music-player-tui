// Package main: mediaremote-adapter (perl + framework) for macOS 15.4+ Now Playing.
// See https://github.com/ungive/mediaremote-adapter

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	perlPath     = "/usr/bin/perl"
	scriptName   = "mediaremote-adapter.pl"
	frameworkDir = "MediaRemoteAdapter.framework"
)

// adapterNowPlaying matches JSON from adapter "get" (optionally with --now).
type adapterNowPlaying struct {
	Title           *string  `json:"title"`
	Artist          *string  `json:"artist"`
	Album           *string  `json:"album"`
	Duration        *float64 `json:"duration"`
	ElapsedTime     *float64 `json:"elapsedTime"`
	ElapsedTimeNow  *float64 `json:"elapsedTimeNow"` // present when "get --now" is used
}

// adapterStreamLine is one line from "stream" (full object or diff).
type adapterStreamLine struct {
	Type    string           `json:"type"`
	Payload adapterNowPlaying `json:"payload"`
}

var (
	adapterBaseDir string
	adapterOnce    sync.Once
)

func findAdapterDir() string {
	adapterOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			return
		}
		dir := filepath.Dir(exe)
		scriptPath := filepath.Join(dir, "mediaremote-adapter", scriptName)
		frameworkPath := filepath.Join(dir, "mediaremote-adapter", frameworkDir)
		if pathExists(scriptPath) && pathExists(frameworkPath) {
			adapterBaseDir = filepath.Join(dir, "mediaremote-adapter")
		}
	})
	return adapterBaseDir
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// AdapterAvailable returns whether mediaremote-adapter is present and usable.
func AdapterAvailable() bool {
	return findAdapterDir() != ""
}

func adapterGet() (model, error) {
	dir := findAdapterDir()
	if dir == "" {
		return model{}, fmt.Errorf("adapter not found")
	}
	scriptPath := filepath.Join(dir, scriptName)
	frameworkPath := filepath.Join(dir, frameworkDir)

	cmd := exec.Command(perlPath, scriptPath, frameworkPath, "get", "--now")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return model{}, fmt.Errorf("adapter get: %w (stderr: %s)", err, string(ee.Stderr))
		}
		return model{}, fmt.Errorf("adapter get: %w", err)
	}

	var raw adapterNowPlaying
	if err := json.Unmarshal(out, &raw); err != nil {
		return model{}, fmt.Errorf("adapter json: %w", err)
	}

	m := model{}
	if raw.Title != nil {
		m.title = *raw.Title
	}
	if m.title == "" {
		m.title = "-"
	}
	if raw.Artist != nil {
		m.artist = *raw.Artist
	}
	if m.artist == "" {
		m.artist = "-"
	}
	if raw.Album != nil {
		m.album = *raw.Album
	}
	if m.album == "" {
		m.album = "-"
	}
	if raw.Duration != nil {
		m.duration = *raw.Duration
	}
	if raw.ElapsedTimeNow != nil {
		m.position = *raw.ElapsedTimeNow
	} else if raw.ElapsedTime != nil {
		m.position = *raw.ElapsedTime
	}
	return m, nil
}

func adapterSendCommand(commandID int) error {
	dir := findAdapterDir()
	if dir == "" {
		return fmt.Errorf("adapter not found")
	}
	scriptPath := filepath.Join(dir, scriptName)
	frameworkPath := filepath.Join(dir, frameworkDir)

	cmd := exec.Command(perlPath, scriptPath, frameworkPath, "send", fmt.Sprintf("%d", commandID))
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// Adapter stream: kMRMediaRemoteCommandTogglePlayPause=2, Next=4, Prev=5
const (
	cmdTogglePlayPause = 2
	cmdNextTrack       = 4
	cmdPreviousTrack   = 5
)

func adapterStartStream(debounceMs int, onLine func(model)) (stop func(), err error) {
	dir := findAdapterDir()
	if dir == "" {
		return nil, fmt.Errorf("adapter not found")
	}
	scriptPath := filepath.Join(dir, scriptName)
	frameworkPath := filepath.Join(dir, frameworkDir)

	args := []string{scriptPath, frameworkPath, "stream", fmt.Sprintf("--debounce=%d", debounceMs)}
	cmd := exec.Command(perlPath, args...)
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 256)
		for {
			n, err := stdout.Read(tmp)
			if err != nil {
				return
			}
			buf = append(buf, tmp[:n]...)
			for {
				idx := strings.IndexByte(string(buf), '\n')
				if idx < 0 {
					break
				}
				line := strings.TrimSpace(string(buf[:idx]))
				buf = buf[idx+1:]
				if line == "" {
					continue
				}
				var event adapterStreamLine
				if json.Unmarshal([]byte(line), &event) != nil {
					continue
				}
				if event.Type != "data" {
					continue
				}
				p := event.Payload
				m := model{title: "-", artist: "-", album: "-"}
				if p.Title != nil {
					m.title = *p.Title
				}
				if p.Artist != nil {
					m.artist = *p.Artist
				}
				if p.Album != nil {
					m.album = *p.Album
				}
				if p.Duration != nil {
					m.duration = *p.Duration
				}
				if p.ElapsedTime != nil {
					m.position = *p.ElapsedTime
				}
				onLine(m)
			}
		}
	}()

	stop = func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
	return stop, nil
}
