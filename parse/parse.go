package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	u, err := url.Parse(apiEndpoint)
	if err != nil {
		return fmt.Errorf("Failed to parse Radio API Endpoint: %v", err)
	}

	q := u.Query()
	q.Set("codec", "MP3")
	q.Set("lastcheckok", "1")

	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return fmt.Errorf("Failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read response body: %v", err)
	}

	var stations []Station
	if err := json.Unmarshal(body, &stations); err != nil {
		return fmt.Errorf("Failed to unmarshal JSON data: %v", err)
	}

	data, err := json.MarshalIndent(stations, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal data into JSON: %v", err)
	}

	filePath := "stations.json"
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("Failed to write data to %s: %v", filePath, err)
	}

	fmt.Println("Data has been successfully written to stations.json")
	fmt.Println("Start the search again, please")

	return nil
}

func FindStation(userTag string) (*Station, error) {
	f, err := os.ReadFile("stations.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("Stations.json not found")
		} else if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("Permission issue with stations.json or its containing directory: %v", err)
		} else {
			return nil, err
		}
	}

	var stations []Station
	if err := json.Unmarshal(f, &stations); err != nil {
		fmt.Println("Remaking the stations.json file")

		if err != nil {
			return nil, fmt.Errorf("Error in stations.json: %v\n", err)
		}

	}

	s := Station{URL: "", Name: "", Tags: ""}
	for _, st := range stations {
		if strings.Contains(unify(st.Tags), unify(userTag)) {
			s = Station{URL: st.URL, Name: st.Name, Tags: st.Tags}
			return &s, nil
		}
	}

	return &s, nil // If no matching station is found, return an empty station
}
