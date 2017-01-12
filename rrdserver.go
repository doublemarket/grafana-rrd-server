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

	"github.com/BurntSushi/toml"
	"github.com/ziutek/rrd"
)

var config Config

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

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	RrdPath string
	Step    int
	Port    int
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
	err := filepath.Walk(config.Server.RrdPath,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() || !strings.Contains(info.Name(), ".rrd") {
				return nil
			}
			rel, _ := filepath.Rel(config.Server.RrdPath, path)
			fName := strings.Replace(rel, ".rrd", "", 1)
			fName = strings.Replace(fName, "/", ":", -1)
			result = append(result, fName)

			return nil
		})
	if err != nil {
		fmt.Println(err)
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
		fPath := config.Server.RrdPath + strings.Replace(target.Target, ":", "/", -1) + ".rrd"
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
		fetchRes, err := rrd.Fetch(fPath, "AVERAGE", from, to, time.Duration(config.Server.Step)*time.Second)
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
	_, err := toml.DecodeFile("config.toml", &config)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/search", search)
	http.HandleFunc("/query", query)
	http.HandleFunc("/annotations", annotations)
	http.HandleFunc("/", hello)

	http.ListenAndServe(":8810", nil)
}
