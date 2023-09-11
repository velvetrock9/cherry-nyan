package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Station struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type model struct {
	controls     []string
	cursor       int
	playing      bool
	streamer     beep.StreamSeekCloser
	radioStation Station
	textInput    textinput.Model
	searching    bool
	errorMessage string
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter search tag( rock / metal / pop / space / jungle / etc)"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		controls: []string{"Play", "Search", "Exit"},
		playing:  false,
		radioStation: Station{
			URL:  "https://rautemusik-de-hz-fal-stream15.radiohost.de/12punks?ref=radiobrowser",
			Name: "12 punks (default)",
		},
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func getStationURL(stationTag string) (string, string, error) {

	baseURL := "http://all.api.radio-browser.info/json/stations/search"

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", "", err
	}

	q := u.Query()
	q.Set("codec", "MP3")
	q.Set("lastcheckok", "1")
	q.Set("tag", stationTag)

	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var stations []Station

	err = json.Unmarshal(body, &stations)
	if err != nil {
		return "", "", err
	}

	if len(stations) > 0 {
		n := rand.Intn(len(stations))
		return stations[n].URL, stations[n].Name, nil
	}
	return "", "", err
}

func connectRadio(stationUrl string) (beep.StreamSeekCloser, error) {

	stream, err := http.Get(stationUrl)
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

	return streamer, nil

}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// When in Search mode, turn j and k controls into actual letters again
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
				m.radioStation.URL, m.radioStation.Name, err = getStationURL(tag)
				if err != nil || m.radioStation.URL == "" {
					m.errorMessage = ("Error: response given by radio API is empty or error")
					return m, nil
				}

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

	// Handle normal controls when NOT in searching mode
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

func main() {

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
