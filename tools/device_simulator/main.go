package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Loc struct {
	BusID    string  `json:"busId"`
	Latitude float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64  `json:"timestamp"`
	SpeedKph  float64 `json:"speedKph"`
	Heading   float64 `json:"heading"`
}

func simulate(i int, wg *sync.WaitGroup, endpoint string) {
	defer wg.Done()
	busid := fmt.Sprintf("bus-%d", i)
	lat, lon := 19.0+float64(i%10)*0.001, 72.0+float64(i%10)*0.001
	for {
		loc := Loc{BusID: busid, Latitude: lat, Longitude: lon, Timestamp: time.Now().Unix(), SpeedKph: 30.0, Heading: 0.0}
		b, _ := json.Marshal(loc)
		http.Post(endpoint, "application/json", bytes.NewReader(b))
		time.Sleep(5 * time.Second)
	}
}

func main() {
	num := 100
	endpoint := "http://localhost:8080/api/locations"
	var wg sync.WaitGroup
	for i := 0; i < num; i++ {
		wg.Add(1)
		go simulate(i, &wg, endpoint)
	}
	wg.Wait()
}
