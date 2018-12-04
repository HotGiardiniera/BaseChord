package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// METRICSFILE represents where we will write our JSON metrics
const METRICSFILE = "/tmp/node%v.JSON"

//JSONTime -> type to easily marshal to proper json time
type JSONTime time.Time

// MarshalJSON for formatting time
func (t JSONTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02T15:04:05.999999"))
	return []byte(stamp), nil
}

//RequestMetric -> object that we will use in chord to gather metrics on requests
type RequestMetric struct {
	SourceNode uint64
	DestNode   uint64
	FileID     uint64
	Hops       uint32
	Start      JSONTime
	End        JSONTime
}

// gatherMetrics will act as a seperate thread to perodically read metric objects inserted from
// our main chord server and process
// Take a channel of request metrics and appends them to a JSON file
func gatherMetrics(metricCh <-chan RequestMetric, nodeID uint64) {
	filename := fmt.Sprintf(METRICSFILE, nodeID)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf(red("%v"), err)
	}
	defer file.Close()
	var metric RequestMetric
	for len(metricCh) > 0 {
		metric = <-metricCh
		blob, err := json.Marshal(metric)
		if err != nil {
			log.Printf(red("%v"), err)
			break
		}
		if _, err = file.WriteString(string(blob) + "\n"); err != nil {
			log.Printf(red("%v"), err)
		}
	}
}
