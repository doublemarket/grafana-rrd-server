package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ziutek/rrd"
)

const (
	step      = 10
	heartbeat = 2 * step
)

type QueryResponse struct {
	Target     string      `json:"target"`
	DataPoints [][]float64 `json:"datapoints"`
}

type QueryRequest struct {
	PanelId int `json:"panelId"`
	Range   struct {
		From string `json:"from"`
		To   string `jsong:"to"`
		Raw  struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"raw"`
	} `json:"range"`
	RangeRaw struct {
		From string `json:"from"`
		To   string `jsong:"to"`
	} `json:"rangeRaw"`
	Interval   string `json:"interval"`
	IntervalMs int    `json:"intervalMs"`
	Targets    []struct {
		Target string `json:"target"`
		RefId  string `json:"refId"`
	} `json:"targets"`
	Format        string `json:"format"`
	MaxDataPoints int    `json:"maxDataPoints"`
}

type Temp struct {
	Message string `json:"message"`
}

func hello(w http.ResponseWriter, r *http.Request) {
	result := Temp{Message: "hello"}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
	json, _ := json.Marshal(result)
	w.Write([]byte(json))
}

func search(w http.ResponseWriter, r *http.Request) {
	var result []string
	directories, _ := filepath.Glob(os.Args[1] + "*")
	for _, d := range directories {
		dName := filepath.Base(d)
		files, _ := filepath.Glob(d + "/*.rrd")
		for _, f := range files {
			fName := strings.Replace(filepath.Base(f), ".rrd", "", 1)
			result = append(result, dName+":"+fName)
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
	json, _ := json.Marshal(result)
	w.Write([]byte(json))
}

func query(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
		w.Write(nil)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var queryRequest QueryRequest
	err := decoder.Decode(&queryRequest)
	if err != nil {
		fmt.Println("error in query 1")
		fmt.Println(err)
	}
	defer r.Body.Close()

	from, _ := time.Parse(time.RFC3339Nano, queryRequest.Range.From)
	to, _ := time.Parse(time.RFC3339Nano, queryRequest.Range.To)

	var result []QueryResponse
	for _, target := range queryRequest.Targets {
		var points [][]float64
		splitTarget := strings.Split(target.Target, ":")
		fPath := os.Args[1] + splitTarget[0] + "/" + splitTarget[1] + ".rrd"
		infoRes, err := rrd.Info(fPath)
		if err != nil {
			fmt.Println("error in query 2")
			fmt.Println(err)
		}
		lastUpdate := time.Unix(int64(infoRes["last_update"].(uint)), 0)
		fmt.Println(from, " ", to, " ", lastUpdate)
		if to.After(lastUpdate) && lastUpdate.After(from) {
			to = lastUpdate
		}
		fmt.Println(from, " ", to, " ", lastUpdate)
		fetchRes, err := rrd.Fetch(fPath, "AVERAGE", from, to, step*time.Second)
		if err != nil {
			fmt.Println("error in query 3")
			fmt.Println(err)
		}
		timestamp := fetchRes.Start
		for _, value := range fetchRes.Values() {
			if math.IsNaN(value) {
				value = 0
			}
			points = append(points, []float64{value, float64(timestamp.Unix()) * 1000})
			timestamp = timestamp.Add(fetchRes.Step)
		}
		defer fetchRes.FreeValues()

		result = append(result, QueryResponse{Target: target.Target, DataPoints: points})
	}
	json, err := json.Marshal(result)
	if err != nil {
		fmt.Println("error when json.Marshal")
		fmt.Println(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
	w.Write([]byte(json))
}

// no need to support
func annotations(w http.ResponseWriter, r *http.Request) {
	result := Temp{Message: "annotations"}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
	json, _ := json.Marshal(result)
	w.Write([]byte(json))
}

func main() {
	http.HandleFunc("/search", search)
	http.HandleFunc("/query", query)
	http.HandleFunc("/annotations", annotations)
	http.HandleFunc("/", hello)

	http.ListenAndServe(":8810", nil)
}
