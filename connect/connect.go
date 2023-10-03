package connect

import (
	"log"
	"net/http"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

// Connects to chosen radio station http stream
func ConnectRadio(url string, connect bool) (beep.StreamSeekCloser, error) {

	stream, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(stream.Body)
	if err != nil {
		log.Fatal(err)
	}

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if connect {
		speaker.Play(streamer)
	} else {
		stream.Body.Close()
	}
	return streamer, nil

}
