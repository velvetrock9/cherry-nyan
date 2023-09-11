package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"log"
	"net/http"
	"os"
	"time"
)

type model struct {
	controls    []string
	cursor      int
	radioSwitch bool
	streamer    beep.StreamSeekCloser
}

func initialModel() model {
	return model{
		controls:    []string{"Play", "Exit"},
		radioSwitch: false,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func connectRadio() (beep.StreamSeekCloser, error) {
	url := "http://stream.bestfm.sk/128.mp3"

	stream, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(stream.Body)
	if err != nil {
		log.Fatal(err)
	}

	//	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	speaker.Play(streamer)
	//select {}
	return streamer, nil

}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.controls)-1 {
				m.cursor++
			}
		case "enter", " ":
			if m.cursor == 1 {
				return m, tea.Quit
			}

			if m.radioSwitch {
				// Stop the radio
				if m.streamer != nil {
					m.streamer.Close()
					m.streamer = nil
					m.radioSwitch = false
				}
			} else {
				streamer, err := connectRadio()
				if err != nil {
					// Handle the error
					log.Fatal(err)
				}
				m.streamer = streamer
				m.radioSwitch = true
			}

		}
	}
	return m, nil
}

func (m model) View() string {
	s := ""
	for i, control := range m.controls {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		s += fmt.Sprintf("%s %s\n", cursor, control)
	}
	s += "\nPress q to quit.\n"
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
