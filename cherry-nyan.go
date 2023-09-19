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
	searching    bool
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
	ti := textinput.New()
	ti.Placeholder = "Enter search tag( rock / metal / pop / space / jungle / etc)"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		controls: []string{"Play", "Search", "Exit"},
		playing:  false,
		radioStation: parse.Station{
			URL:  "https://rautemusik-de-hz-fal-stream15.radiohost.de/12punks?ref=radiobrowser",
			Name: "12 punks (default)",
			Tags: "punk",
		},
		textInput: ti,
		songTitle: "InitialModel Song Title",
	}
}

func (m model) Init() tea.Cmd {
	return doTick(m.radioStation.URL)
}

// Update model state. Mostly describes logic which happens during keyboard events.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	/*
	   Though not DRY, this code block is required, until concurrency implementation,
	   so the Search context could react on Enter and not interpret j k keys as controls.
	*/

	if m.searching {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":

				// Handling the Enter key to complete the search
				tag := m.textInput.Value()
				if tag == "" {
					m.errorMessage = "Error: response given by radio API is empty or error"
					return m, nil
				}

				var err error
				station := parse.FindStation(tag)

				if err != nil || station.URL == "" {
					m.errorMessage = fmt.Sprintf("Error: no station with a tag %v", tag)
					return m, nil
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
				m.playing = true

				// After processing the tag, reset the input and hide it.
				m.textInput.SetValue("")
				m.searching = false

				return m, nil
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}
	}

	// Handle normal controls when NOT in a Search mode
	switch msg := msg.(type) {
	case TickMsg:
		m.songTitle = string(msg)
		return m, doTick(m.radioStation.URL)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j", "ctrl+n":
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
	for i, control := range m.controls {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		if i == 0 {
			if m.playing {
				control = "Pause"
			} else {
				control = "Play"
			}
		}

		s += fmt.Sprintf("%s %s\n", cursor, control)
	}
	if m.playing {
		s += fmt.Sprintf("\nNow Playing: %s\n", m.radioStation.Name)
		s += fmt.Sprintf("SongTitle: %s", m.songTitle)
	}
	s += "\nPress Exit or q to quit.\n"
	if m.searching {
		s += "\n" + m.textInput.View()
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
