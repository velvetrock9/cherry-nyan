package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const apiEndpoint = "http://all.api.radio-browser.info/json/stations/search"

type Station struct {
	Changeuuid      string   `json:"changeuuid"`
	Stationuuid     string   `json:"stationuuid"`
	Serveruuid      string   `json:"serveruuid"`
	Name            string   `json:"name"`
	URL             string   `json:"url"`
	URLResolved     string   `json:"url_resolved"`
	Homepage        string   `json:"homepage"`
	Favicon         string   `json:"favicon"`
	Tags            string   `json:"tags"`
	Country         string   `json:"country"`
	CountryCode     string   `json:"countrycode"`
	Iso31662        string   `json:"iso_3166_2"`
	State           string   `json:"state"`
	Language        string   `json:"language"`
	LanguageCodes   string   `json:"languagecodes"`
	Votes           int      `json:"votes"`
	LastChangeTime  string   `json:"lastchangetime"`
	Codec           string   `json:"codec"`
	Bitrate         int      `json:"bitrate"`
	HLS             int      `json:"hls"`
	LastCheckOK     int      `json:"lastcheckok"`
	ClickCount      int      `json:"clickcount"`
	ClickTrend      int      `json:"clicktrend"`
	SSLError        int      `json:"ssl_error"`
	GeoLat          *float64 `json:"geo_lat"`
	GeoLong         *float64 `json:"geo_long"`
	HasExtendedInfo bool     `json:"has_extended_info"`
}

func unify(input string) string {
	s := ""
	s = strings.TrimSpace(input)
	s = strings.ToLower(s)
	return s
}

func ParseStations() error {

	// Processing baseURL  and query arguments through url package for safety reasons
	u, err := url.Parse(apiEndpoint)
	if err != nil {
		log.Fatalf("Failed to parse Radio API Endpoint: %v", err)
	}

	q := u.Query()
	q.Set("codec", "MP3")
	q.Set("lastcheckok", "1")

	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	var stations []Station
	if err := json.Unmarshal(body, &stations); err != nil {
		log.Fatalf("Failed to unmarshal JSON data: %v", err)
	}

	data, err := json.MarshalIndent(stations, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal data into JSON: %v", err)
	}

	filePath := "stations.json"
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		log.Fatalf("Failed to write data to %s: %v", filePath, err)
	}

	log.Println("Data successfully written to stations.json")

	return nil
}

func FindStation(userTag string) *Station {

	f, err := os.ReadFile("stations.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ParseStations()

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

	var stations []Station

	if err := json.Unmarshal(f, &stations); err != nil {
		log.Fatalf("Failed to unmarshal JSON data: %v", err)
	}
	s := Station{URL: "", Name: "", Tags: ""}
	for _, st := range stations {
		if strings.Contains(unify(st.Tags), unify(userTag)) {
			s = Station{URL: st.URL, Name: st.Name, Tags: st.Tags}
			return &s
		}
	}
	return &s
}
