package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -F/System/Library/PrivateFrameworks -framework Foundation -framework MediaRemote
#include "mediaremote.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type changeMsg struct{}
type tickMsg struct{}

var changeChan = make(chan struct{})

// useAdapter is set at startup; when true, we use mediaremote-adapter (perl) instead of C MediaRemote.
var useAdapter bool

//export mr_on_change
func mr_on_change() {
	changeChan <- struct{}{}
}

type model struct {
	title    string
	artist   string
	album    string
	position float64
	duration float64
}

func fetch() model {
	if useAdapter {
		m, err := adapterGet()
		if err != nil {
			return model{title: "-", artist: "-", album: "-"}
		}
		return m
	}
	C.mr_refresh()
	title := C.GoString(C.mr_title())
	artist := C.GoString(C.mr_artist())
	album := C.GoString(C.mr_album())
	pos := float64(C.mr_position())
	dur := float64(C.mr_duration())
	return model{
		title:    title,
		artist:   artist,
		album:    album,
		position: pos,
		duration: dur,
	}
}

func waitChange() tea.Cmd {
	return func() tea.Msg {
		<-changeChan
		return changeMsg{}
	}
}

func (m model) Init() tea.Cmd {
	tick := func() tea.Cmd {
		return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
	}
	if useAdapter {
		stop, err := adapterStartStream(300, func(_ model) {
			changeChan <- struct{}{}
		})
		if err == nil {
			_ = stop
		}
		return tea.Batch(waitChange(), tick())
	}
	C.mr_start_listener()
	return tea.Batch(waitChange(), tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case changeMsg:
		return fetch(), waitChange()

	case tickMsg:
		return fetch(), tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })

	case tea.KeyMsg:

		switch msg.String() {

		case "q", "ctrl+c":
			return m, tea.Quit

		case " ":
			if useAdapter {
				_ = adapterSendCommand(cmdTogglePlayPause)
			} else {
				C.mr_play_pause()
			}
		case "n":
			if useAdapter {
				_ = adapterSendCommand(cmdNextTrack)
			} else {
				C.mr_next()
			}
		case "p":
			if useAdapter {
				_ = adapterSendCommand(cmdPreviousTrack)
			} else {
				C.mr_prev()
			}
		}
	}

	return m, nil
}

func formatTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	m := int(sec) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func progress(pos, dur float64) string {
	width := 20
	if dur <= 0 {
		dur = 1
	}
	p := int((pos / dur) * float64(width))
	if p > width {
		p = width
	}

	bar := ""

	for i := 0; i < width; i++ {
		if i < p {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return bar
}

func (m model) View() string {

	return fmt.Sprintf(
		`
now playing

Artist: %s
Title : %s
Album : %s

%s [%s] %s

space: play/pause
n: next   p: prev
q: quit
`,
		m.artist,
		m.title,
		m.album,
		formatTime(m.position),
		progress(m.position, m.duration),
		formatTime(m.duration),
	)
}

func main() {
	useAdapter = AdapterAvailable()

	p := tea.NewProgram(fetch())

	if err := p.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}