package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/faiface/beep"
	"github.com/velvetrock9/cherry-nyan/connect"
	"github.com/velvetrock9/cherry-nyan/icy"
	"github.com/velvetrock9/cherry-nyan/parse"
)

type TickMsg string
type sessionState uint

type model struct {
	state        sessionState
	control      string
	cursor       int
	streamer     beep.StreamSeekCloser
	radioStation parse.Station
	textInput    textinput.Model
	errorMessage string
	songTitle    string
	spinner      spinner.Model
	isPlaying    bool
}

const (
	generalView sessionState = iota
	searchView
)

var (
	helpStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#000AAA")).
			Foreground(lipgloss.Color("241")).
			Width(50)
	searchStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("123")).
			Width(50)

	radioStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#083D77")).
			Foreground(lipgloss.Color("#EBEBD3")).
			Width(50)
)

func doTick(radioStation string, condition bool) tea.Cmd {
	return tea.Every(time.Second*1, func(t time.Time) tea.Msg {
		if condition {
			message, err := icy.GrabSongTitle(radioStation)
			if err != nil {
				message = fmt.Sprintf("Error: %v", err)
			}
			return TickMsg(message)
		}
		message := "Condition `isPlaying=True` hasn't been met"
		return TickMsg(message)
	})
}

// Initial model state
func newModel() model {
	m := model{state: generalView}

	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m.textInput = textinput.New()
	m.textInput.Placeholder = "Enter search tag(rock / metal / pop / space / jungle / etc)"
	m.textInput.PlaceholderStyle = searchStyle
	m.textInput.Focus()
	m.textInput.CharLimit = 156
	m.textInput.Width = 20

	m.isPlaying = false

	m.radioStation = parse.Station{
		URL:  "https://rautemusik-de-hz-fal-stream15.radiohost.de/12punks?ref=radiobrowser",
		Name: "12 punks (default)",
		Tags: "punk",
	}
	return m

}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.state == searchView {
				m.textInput.SetValue("")
				m.state = generalView
			} else if m.state == generalView {
				m.textInput.SetValue("")
				m.state = searchView
			}
		case "q", "esc":
			if m.state == searchView {
				m.textInput.SetValue("")
				m.state = generalView
			} else if m.state == generalView {
				m.textInput.SetValue("")
				return m, tea.Quit
			}
		case "ctrl+c":
			m.textInput.SetValue("")
			return m, tea.Quit
		case "enter", " ":
			if m.state == searchView {
				m.state = generalView
				m.isPlaying = false
				cmds = append(cmds, m.spinner.Tick)
				tag := m.textInput.Value()
				if tag == "" {
					m.errorMessage = "Error: response given by radio API is empty or error"
					fmt.Println(m.errorMessage)
				}

				station, err := parse.FindStation(tag)
				if err != nil {
					fmt.Println(err)
					parse.ParseStations()
					return m, nil

				}

				m.radioStation = *station

				// Stop the radio
				if m.streamer != nil {
					m.streamer.Close()
					m.streamer = nil
					m.songTitle = ""
				}

				m.isPlaying = true
				streamer, err := connect.ConnectRadio(m.radioStation.URL, m.isPlaying)
				if err != nil {
					// Needs refactoring
					fmt.Println("Can't connect to the radio")

				}
				m.streamer = streamer

				m.songTitle = ""
				m.textInput.SetValue("")
				cmds = append(cmds, doTick(m.radioStation.URL, m.isPlaying))

			} else if m.state == generalView {
				// Stop the radio
				if m.isPlaying {
					m.streamer.Close()
					m.streamer = nil
					m.isPlaying = false
					m.songTitle = ""

					// Play the radio
				} else {
					cmds = append(cmds, m.spinner.Tick)
					m.isPlaying = true
					streamer, err := connect.ConnectRadio(m.radioStation.URL, m.isPlaying)
					if err != nil {
						// Handle the error
						fmt.Println("Can't connect to the radio")

					}
					m.streamer = streamer
					// Reset song title after connecting to new station
					cmds = append(cmds, doTick(m.radioStation.URL, m.isPlaying))
					// After processing the tag, reset the input and hide it.
					m.textInput.SetValue("")
				}
			}
		}

		switch m.state {
		case searchView:
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
		}

	case TickMsg:
		var err error
		m.songTitle, err = icy.GrabSongTitle(m.radioStation.URL)
		if err != nil {
			m.songTitle = fmt.Sprintf("Error: %v", err)
		}

		cmds = append(cmds, doTick(m.radioStation.URL, m.isPlaying))

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)

}

// Describes View logic.
func (m model) View() string {
	var s string

	if m.isPlaying {
		m.control = "⏸️ Pause"
	} else {
		m.control = "▶️ Play"
		m.radioStation.Name = " "
		m.songTitle = " "

	}

	s = "\n"
	s += radioStyle.Render(
		fmt.Sprintf("📻 Radio: %s\n", m.radioStation.Name))
	s += "\n"
	s += radioStyle.Render(fmt.Sprintf("📻 isPlaying: %t\n", m.isPlaying))

	s += "\n"
	if m.songTitle == "" {
		s += radioStyle.Render(fmt.Sprintf("🎶 Track: %sLoading", m.spinner.View()))
	} else {
		s += radioStyle.Render(fmt.Sprintf("🎶 Track: %s", m.songTitle))
	}

	if m.state == searchView {
		s += "\n" + searchStyle.Render(m.textInput.View())

	}

	// Renders help string. ALWAYS needs to be rendered.
	s += "\n" + fmt.Sprintf("Tab: Search • Enter/Space: %s • Esc/q: exit\n", m.control)
	return s
}

// Main goroutine
func main() {

	p := tea.NewProgram(newModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
