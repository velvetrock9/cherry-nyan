package icy

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strconv"
)

type TickMsg string

// grab Song title from metadata
func GrabSongTitle(url string) (string, error) {
	m, err := getStreamMetas(url)

	if err != nil {
		return "", fmt.Errorf("Error: error while requesting stream ICY meta")
	}
	// Should be at least "StreamTitle=' '"
	if len(m) < 15 {
		return "", fmt.Errorf("Error: StreamTitle is empty")
	}
	// Split meta by ';', trim it and search for StreamTitle
	for _, m := range bytes.Split(m, []byte(";")) {
		m = bytes.Trim(m, " \t")
		if bytes.Compare(m[0:13], []byte("StreamTitle='")) != 0 {
			continue
		}
		return string(m[13 : len(m)-1]), nil
	}

	return `¯\_(ツ)_/¯`, fmt.Errorf("Empty return")
}

// get stream metadatas
func getStreamMetas(streamUrl string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", streamUrl, nil)
	req.Header.Set("Icy-MetaData", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// We sent "Icy-MetaData", we should have a "icy-metaint" in return
	ih := resp.Header.Get("icy-metaint")
	if ih == "" {
		return nil, fmt.Errorf("We didn't get an icy-metaint header")
	}

	// "icy-metaint" is how often (in bytes) should we receive the meta
	ib, err := strconv.Atoi(ih)
	if err != nil {
		return nil, fmt.Errorf("Can't conver't icy-metaint header to bytes")
	}

	reader := bufio.NewReader(resp.Body)

	// skip the first mp3 frame
	c, err := reader.Discard(ib)
	if err != nil {
		return nil, fmt.Errorf("Can't Discard mp3 frame")
	}
	// If we didn't received ib bytes, the stream is ended
	if c != ib {
		return nil, fmt.Errorf("Stream has ended prematurally")
	}

	// get the size byte, that is the metadata length in bytes / 16
	sb, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("Error while getting a size of radio stream metadata")
	}
	ms := int(sb * 16)

	// read the ms first bytes it will contain metadata
	m, err := reader.Peek(ms)
	if err != nil {
		return nil, fmt.Errorf("There should be a metadata in byte stream, but I've found none of it.")
	}
	return m, nil
}
