func getStationURL(stationTag string) *station {
	f := os.ReadFile("/parse-stations.go/stations.json")
	if err != nil {
		log.Fatal("unable to read stations.json")
	}

	var stations []station

	if err := json.Unmarshal(f, &stations); err != nil {
		log.Fatalf("Failed to unmarshal JSON data: %v", err)
	}

	for i := 0; i < 5; i++ {
		fmt.Println(stations[i])
	}

}
