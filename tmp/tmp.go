package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"time"
)

type TickMsg time.Time

type model struct {
	time string
}

func initialModel() model {
	return model{time: "start"}
}

// Get stream metadata
func getStreamMetas(streamUrl string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", streamUrl, nil)
	req.Header.Set("Icy-MetaData", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// We sent "Icy-MetaData", we should have an "icy-metaint" in return
	ih := resp.Header.Get("icy-metaint")
	if ih == "" {
		return nil, fmt.Errorf("no metadata")
	}
	// "icy-metaint" is how often (in bytes) we should receive the meta
	ib, err := strconv.Atoi(ih)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(resp.Body)

	// Skip the first MP3 frame
	c, err := reader.Discard(ib)
	if err != nil {
		return nil, err
	}
	// If we didn't receive ib bytes, the stream ended prematurely
	if c != ib {
		return nil, fmt.Errorf("stream ended prematurely")
	}

	// Get the size byte, which is the metadata length in bytes / 16
	sb, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	ms := int(sb * 16)

	// Read the ms first bytes; it will contain metadata
	m, err := reader.Peek(ms)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func fetchTrackTitle() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		trackTitle := ""
		m, err := getStreamMetas(streamUrl)

		if err != nil {
			return "", err
		}
		// Should be at least "StreamTitle=' '"
		if len(m) < 15 {
			return "", nil
		}
		// Split meta by ';', trim it and search for StreamTitle
		for _, m := range bytes.Split(m, []byte(";")) {
			m = bytes.Trim(m, " \t")
			if bytes.Compare(m[0:13], []byte("StreamTitle='")) != 0 {
				continue
			}
			trackTitle = string(m[13 : len(m)-1]), nil
		}
		return trackTitle, nil
	})
}

func (m model) Init() tea.Cmd {
	// Start ticking.
	return doTick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		t := time.Time(msg.(TickMsg))
		m.time = t.Format(time.RubyDate)
		return m, doTick()
	case tea.KeyMsg:
		return m.handleKey(msg.(tea.KeyMsg))
	}
	return m, nil
}

func (m model) handleKey(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyMsg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	s := fmt.Sprintf("Time: %s", m.time)
	return s
}

func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
