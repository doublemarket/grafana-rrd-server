package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
		RefID  string `json:"refId"`
		Hide   bool   `json:"hide"`
		Type   string `json:"type"`
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

type ErrorResponse struct {
	Message string `json:"message"`
}

func hello(w http.ResponseWriter, r *http.Request) {
	result := ErrorResponse{Message: "hello"}
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
			infoRes, err := rrd.Info(path)
			if err != nil {
				fmt.Println("ERROR: Cannot retrieve information from ", path)
				fmt.Println(err)
			}
			for ds, _ := range infoRes["ds.index"].(map[string]interface{}) {
				result = append(result, fName+":"+ds)
			}

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
		fmt.Println("ERROR: Cannot decode the request")
		fmt.Println(err)
	}
	defer r.Body.Close()

	from, _ := time.Parse(time.RFC3339Nano, queryRequest.Range.From)
	to, _ := time.Parse(time.RFC3339Nano, queryRequest.Range.To)

	var result []QueryResponse
	for _, target := range queryRequest.Targets {
		var points [][]float64
		ds := target.Target[strings.LastIndex(target.Target, ":")+1 : len(target.Target)]
		rrdDsRep := regexp.MustCompile(`:` + ds + `$`)
		fPath := rrdDsRep.ReplaceAllString(target.Target, "")
		fPath = config.Server.RrdPath + strings.Replace(fPath, ":", "/", -1) + ".rrd"
		if _, err := os.Stat(fPath); err != nil {
			fmt.Println("File", fPath, "does not exist")
			continue
		}
		infoRes, err := rrd.Info(fPath)
		if err != nil {
			fmt.Println("ERROR: Cannot retrieve information from ", fPath)
			fmt.Println(err)
		}
		lastUpdate := time.Unix(int64(infoRes["last_update"].(uint)), 0)
		if to.After(lastUpdate) && lastUpdate.After(from) {
			to = lastUpdate
		}
		fetchRes, err := rrd.Fetch(fPath, "AVERAGE", from, to, time.Duration(config.Server.Step)*time.Second)
		if err != nil {
			fmt.Println("ERROR: Cannot retrieve time series data from ", fPath)
			fmt.Println(err)
		}
		timestamp := fetchRes.Start
		dsIndex := int(infoRes["ds.index"].(map[string]interface{})[ds].(uint))
		// The last point is likely to contain wrong data (mostly a big number)
		// RowCnt-1 is for ignoring the last point (temporary solution)
		for i := 0; i < fetchRes.RowCnt-1; i++ {
			value := fetchRes.ValueAt(dsIndex, i)
			if !math.IsNaN(value) {
				points = append(points, []float64{value, float64(timestamp.Unix()) * 1000})
			}
			timestamp = timestamp.Add(fetchRes.Step)
		}
		defer fetchRes.FreeValues()

		result = append(result, QueryResponse{Target: target.Target, DataPoints: points})
	}
	json, err := json.Marshal(result)
	if err != nil {
		fmt.Println("ERROR: Cannot convert response data into JSON")
		fmt.Println(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
	w.Write([]byte(json))
}

func annotations(w http.ResponseWriter, r *http.Request) {
	result := ErrorResponse{Message: "annotations"}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
	json, _ := json.Marshal(result)
	w.Write([]byte(json))
}

func ReadConfigFile(filename string) error {
	_, err := toml.DecodeFile(filename, &config)
	return err
}

func main() {
	configFilePath := "config.toml"
	err := ReadConfigFile(configFilePath)
	if err != nil {
		fmt.Println("ERROR: Cannot read config file ", configFilePath)
		panic(err)
	}

	http.HandleFunc("/search", search)
	http.HandleFunc("/query", query)
	http.HandleFunc("/annotations", annotations)
	http.HandleFunc("/", hello)

	http.ListenAndServe(":"+strconv.Itoa(config.Server.Port), nil)
}
