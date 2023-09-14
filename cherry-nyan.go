package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/velvetrock9/cherry-nyan/parse"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type station struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Tag  string `json:"tags"`
}

type model struct {
	controls     []string
	cursor       int
	playing      bool
	streamer     beep.StreamSeekCloser
	radioStation station
	textInput    textinput.Model
	searching    bool
	errorMessage string
}

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
		radioStation: station{
			URL:  "https://rautemusik-de-hz-fal-stream15.radiohost.de/12punks?ref=radiobrowser",
			Name: "12 punks (default)",
			Tag:  "punk",
		},
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func unify(input string) string {
	s := ""
	s = strings.TrimSpace(input)
	s = strings.ToLower(s)
	return s
}

func findStation(userTag string) *station {

	f, err := os.ReadFile("stations.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			parse.ParseStations()

		} else if errors.Is(err, os.ErrPermission) {
			fmt.Errorf(`something is wrong with your stations.json
permissions or its containing directory`)
		} else {
			panic(err)
		}
	}
	if err != nil {
		log.Fatal("unable to read stations.json")
	}

	var stations []station

	if err := json.Unmarshal(f, &stations); err != nil {
		log.Fatalf("Failed to unmarshal JSON data: %v", err)
	}
	s := station{URL: "", Name: "", Tag: ""}
	for _, st := range stations {
		if strings.Contains(unify(st.Tag), unify(userTag)) {
			s = station{URL: st.URL, Name: st.Name, Tag: st.Tag}
			fmt.Println(s)
			return &s
		}
	}
	return &s
}

// Connects to chosen radio station http stream
func connectRadio(url string) (beep.StreamSeekCloser, error) {

	stream, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(stream.Body)
	if err != nil {
		log.Fatal(err)
	}

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	speaker.Play(streamer)

	return streamer, nil

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
				station := findStation(tag)

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

				streamer, err := connectRadio(m.radioStation.URL)
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
				streamer, err := connectRadio(m.radioStation.URL)
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
