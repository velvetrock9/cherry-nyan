package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
	"github.com/velvetrock9/cherry-nyan/connect"
	"github.com/velvetrock9/cherry-nyan/icy"
	"github.com/velvetrock9/cherry-nyan/parse"
	"log"
	"os"
	"time"
)

type model struct {
	controls     []string
	cursor       int
	playing      bool
	streamer     beep.StreamSeekCloser
	radioStation parse.Station
	textInput    textinput.Model
	answerInput  textinput.Model
	searching    bool
	asking       bool
	errorMessage string
	songTitle    string
}

func doTick(radioStation string) tea.Cmd {
	return tea.Tick(time.Second*7, func(t time.Time) tea.Msg {

		message, err := icy.GrabSongTitle(radioStation)
		if err != nil {
			fmt.Println(err)
		}
		return TickMsg(message)
	})
}

type TickMsg string

// Initial model state
func initialModel() model {
	answer := textinput.New()
	answer.Placeholder = "Do you want to generate a new stations.json file? (yes/no)"
	answer.Focus()
	answer.CharLimit = 156
	answer.Width = 20

	ti := textinput.New()
	ti.Placeholder = "Enter search tag( rock / metal / pop / space / jungle / etc)"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		controls: []string{"▶️ Play", "🔎 Search", "🔚 Exit"},
		playing:  false,
		radioStation: parse.Station{
			URL:  "https://rautemusik-de-hz-fal-stream15.radiohost.de/12punks?ref=radiobrowser",
			Name: "12 punks (default)",
			Tags: "punk",
		},
		textInput:   ti,
		answerInput: answer,
	}
}

func (m model) Init() tea.Cmd {
	return doTick(m.radioStation.URL)
}

// Update model state. Mostly describes logic which happens during keyboard events.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var tag string
	/*
	   Though not DRY, this code block is required, so the Search context could react on Enter and not react on j k keys as controls.
	*/

	// Search context
	if m.searching {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.searching = false
				fmt.Println("Search context: OFF")
				return m, cmd
			case "enter":
				tag = m.textInput.Value()
				m.searching = false
				// Main logic of finding an appropriate station and connecting to it
				station, err := parse.FindStation(tag)
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println("Rebuilding list of radio stations...")
					parse.ParseStations()
					return m, cmd
				}
				m.radioStation = *station
				// Stop the radio
				if m.streamer != nil {
					m.streamer.Close()
					m.streamer = nil
					m.playing = false
				}

				streamer, err := connect.ConnectRadio(m.radioStation.URL)
				if err != nil {
					// Handle the error
					log.Fatal(err)
				}
				m.streamer = streamer
				// Reset song title after connecting to new station
				m.songTitle = ""
				m.playing = true
				return m, cmd
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

	}

	switch msg := msg.(type) {
	case TickMsg:
		if string(msg) == "" {
			m.songTitle = `¯\_(ツ)_/¯`
		} else {
			m.songTitle = string(msg)
		}
		return m, doTick(m.radioStation.URL)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "left", "h", "ctrl+b":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "l", "ctrl+f":
			if m.cursor < len(m.controls)-1 {
				m.cursor++
			}
		case "enter", " ":
			if m.cursor == 2 { // Exit
				return m, tea.Quit
			}

			if m.cursor == 1 { // Search
				// Switch to search mode
				m.searching = true
				return m, nil
			}

			if m.playing {
				if m.streamer != nil {
					m.streamer.Close()
					m.streamer = nil
					m.playing = false
				}
			} else {
				streamer, err := connect.ConnectRadio(m.radioStation.URL)
				if err != nil {
					log.Fatal(err)
				}
				m.streamer = streamer
				m.playing = true
			}
		}
	}

	return m, cmd
}

// Describes View logic.
func (m model) View() string {
	s := ""
	s += fmt.Sprintf("\n\n")
	for i, control := range m.controls {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		if i == 0 {
			if m.playing {
				control = "⏸️ Pause"
			} else {
				control = "▶️ Play"
			}
		}

		s += fmt.Sprintf("%s %s", cursor, control)
	}
	if m.playing {
		s += fmt.Sprintf("\n\n")
		s += fmt.Sprintf("📻 Radio: %s\n", m.radioStation.Name)
		s += fmt.Sprintf("🎶 Track: %s\n", m.songTitle)
	}
	if m.searching {
		s += "\n" + m.textInput.View()
	}
	if m.asking {
		s += "\n" + m.answerInput.View()
	}
	return s
}

// Main goroutine
func main() {

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
