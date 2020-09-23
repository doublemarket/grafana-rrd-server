package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/mattn/go-zglob"
	"github.com/ziutek/rrd"
)

var config Config

type QueryResponse struct {
	Target     string      `json:"target"`
	DataPoints [][]float64 `json:"datapoints"`
}

type SearchRequest struct {
    Target string `json:"target"`
}

type QueryRequest struct {
	PanelId int64 `json:"panelId"`
	Range   struct {
		From string `json:"from"`
		To   string `json:"to"`
		Raw  struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"raw"`
	} `json:"range"`
	RangeRaw struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"rangeRaw"`
	Interval   string `json:"interval"`
	IntervalMs int64  `json:"intervalMs"`
	Targets    []struct {
		Target string `json:"target"`
		RefID  string `json:"refId"`
		Hide   bool   `json:"hide"`
		Type   string `json:"type"`
	} `json:"targets"`
	Format        string `json:"format"`
	MaxDataPoints int64  `json:"maxDataPoints"`
}

type AnnotationResponse struct {
	Annotation string `json:"annotation"`
	Time       int64  `json:"time"`
	Title      string `json:"title"`
	Tags       string `json:"tags"`
	Text       string `json:"text"`
}

type AnnotationCSV struct {
	Time  int64  `csv:"time"`
	Title string `csv:"title"`
	Tags  string `csv:"tags"`
	Text  string `csv:"text"`
}

type AnnotationRequest struct {
	Range struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"range"`
	RangeRaw struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"rangeRaw"`
	Annotation struct {
		Name       string `json:"name"`
		Datasource string `json:"datasource"`
		IconColor  string `json:"iconColor"`
		Enable     bool   `json:"enable"`
		Query      string `json:"query"`
	} `json:"annotation"`
}

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	RrdPath            string
	Step               int
	SearchCache        int64
	IpAddr             string
	Port               int
	AnnotationFilePath string
	Multiplier         int
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func respondJSON(w http.ResponseWriter, result interface{}) {
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

func hello(w http.ResponseWriter, r *http.Request) {
	result := ErrorResponse{Message: "hello"}
	respondJSON(w, result)
}

var cacheSearch []string
var lastSearchUpdate int64

func search(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var searchRequest SearchRequest
	err := decoder.Decode(&searchRequest)
	if err != nil {
		fmt.Println("ERROR: Cannot decode the request")
		fmt.Println(err)
	}
	defer r.Body.Close()

	target := searchRequest.Target

	now := time.Now().Unix()
	if len(cacheSearch) == 0 || (lastSearchUpdate + config.Server.SearchCache) < now {
		lastSearchUpdate = now
		cacheSearch = []string{}

		fmt.Println("Updating search cache.")

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
					cacheSearch = append(cacheSearch, fName+":"+ds)
				}

				return nil
			})

		if err != nil {
			fmt.Println(err)
		}
	}

	var result = []string{}

	if target != "" {
		for _, path := range cacheSearch {
			if (strings.Contains(path, target)) {
				result = append(result, path)
			}
		}
	}

	respondJSON(w, result)
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
		ds := target.Target[strings.LastIndex(target.Target, ":")+1 : len(target.Target)]
		rrdDsRep := regexp.MustCompile(`:` + ds + `$`)
		fileSearchPath := rrdDsRep.ReplaceAllString(target.Target, "")
		fileSearchPath = strings.TrimRight(config.Server.RrdPath, "/") + "/" + strings.Replace(fileSearchPath, ":", "/", -1) + ".rrd"

		fileNameArray, _ := zglob.Glob(fileSearchPath)
		for _, filePath := range fileNameArray {
			points := make([][]float64, 0)
			if _, err = os.Stat(filePath); err != nil {
				fmt.Println("File", filePath, "does not exist")
				continue
			}
			infoRes, err := rrd.Info(filePath)
			if err != nil {
				fmt.Println("ERROR: Cannot retrieve information from ", filePath)
				fmt.Println(err)
			}
			lastUpdate := time.Unix(int64(infoRes["last_update"].(uint)), 0)
			if to.After(lastUpdate) && lastUpdate.After(from) {
				to = lastUpdate
			}
			fetchRes, err := rrd.Fetch(filePath, "AVERAGE", from, to, time.Duration(config.Server.Step)*time.Second)
			if err != nil {
				fmt.Println("ERROR: Cannot retrieve time series data from ", filePath)
				fmt.Println(err)
			}
			timestamp := fetchRes.Start
			dsIndex := int(infoRes["ds.index"].(map[string]interface{})[ds].(uint))
			// The last point is likely to contain wrong data (mostly a big number)
			// RowCnt-1 is for ignoring the last point (temporary solution)
			for i := 0; i < fetchRes.RowCnt-1; i++ {
				value := fetchRes.ValueAt(dsIndex, i)
				if !math.IsNaN(value) {
					product := float64(config.Server.Multiplier) * value
					points = append(points, []float64{product, float64(timestamp.Unix()) * 1000})
				}
				timestamp = timestamp.Add(fetchRes.Step)
			}
			defer fetchRes.FreeValues()

			extractedTarget := strings.Replace(filePath, ".rrd", "", -1)
			extractedTarget = strings.Replace(extractedTarget, config.Server.RrdPath, "", -1)
			extractedTarget = strings.Replace(extractedTarget, "/", ":", -1) + ":" + ds
			result = append(result, QueryResponse{Target: extractedTarget, DataPoints: points})
		}
	}
	respondJSON(w, result)
}

func annotations(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Write(nil)
		return
	}

	if config.Server.AnnotationFilePath == "" {
		result := ErrorResponse{Message: "Not configured"}
		respondJSON(w, result)
	} else {
		decoder := json.NewDecoder(r.Body)
		var annotationRequest AnnotationRequest
		err := decoder.Decode(&annotationRequest)
		defer r.Body.Close()
		if err != nil {
			result := ErrorResponse{Message: "Cannot decode the request"}
			respondJSON(w, result)
		} else {
			csvFile, err := os.OpenFile(config.Server.AnnotationFilePath, os.O_RDONLY, os.ModePerm)
			if err != nil {
				fmt.Println("ERROR: Cannot open the annotations CSV file ", config.Server.AnnotationFilePath)
				fmt.Println(err)
			}
			defer csvFile.Close()
			annots := []*AnnotationCSV{}

			if err := gocsv.UnmarshalFile(csvFile, &annots); err != nil {
				fmt.Println("ERROR: Cannot unmarshal the annotations CSV file.")
				fmt.Println(err)
			}

			result := []AnnotationResponse{}
			from, _ := time.Parse(time.RFC3339Nano, annotationRequest.Range.From)
			to, _ := time.Parse(time.RFC3339Nano, annotationRequest.Range.To)
			for _, a := range annots {
				if (from.Unix()*1000) <= a.Time && a.Time <= (to.Unix()*1000) {
					result = append(result, AnnotationResponse{Annotation: "annotation", Time: a.Time, Title: a.Title, Tags: a.Tags, Text: a.Text})
				}
			}
			respondJSON(w, result)
		}
	}
}

func SetArgs() {
	flag.StringVar(&config.Server.IpAddr, "i", "", "Network interface IP address to listen on. (default: any)")
	flag.IntVar(&config.Server.Port, "p", 9000, "Server port.")
	flag.StringVar(&config.Server.RrdPath, "r", "./sample/", "Path for a directory that keeps RRD files.")
	flag.IntVar(&config.Server.Step, "s", 10, "Step in second.")
	flag.Int64Var(&config.Server.SearchCache, "c", 600, "Search cache in seconds.")
	flag.StringVar(&config.Server.AnnotationFilePath, "a", "", "Path for a file that has annotations.")
	flag.IntVar(&config.Server.Multiplier, "m", 1, "Value multiplier.")
	flag.Parse()
}

func main() {
	SetArgs()

	http.HandleFunc("/search", search)
	http.HandleFunc("/query", query)
	http.HandleFunc("/annotations", annotations)
	http.HandleFunc("/", hello)

	err := http.ListenAndServe(config.Server.IpAddr+":"+strconv.Itoa(config.Server.Port), nil)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}
