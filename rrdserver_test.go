package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHello(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(hello))
	defer ts.Close()

	r, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("Error by http.Get(). %v", err)
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Error by ioutil.ReadAll(). %v", err)
	}

	if r.StatusCode != 200 {
		t.Fatalf("Status code is not 200 but %d.", r.StatusCode)
	}

	if "{\"message\":\"hello\"}" != string(data) {
		t.Fatalf("Data Error. %v", string(data))
	}
}

func TestSearch(t *testing.T) {
	SetArgs()

	requestJSON := `{"target":"hoge"}`
	reader := strings.NewReader(requestJSON)

	ts := httptest.NewServer(http.HandlerFunc(search))
	defer ts.Close()

	r, err := http.Post(ts.URL, "application/json", reader)
	if err != nil {
		t.Fatalf("Error at a GET request. %v", err)
	}

	if r.StatusCode != 200 {
		t.Fatalf("Status code is not 200 but %d.", r.StatusCode)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var searchResponse []string
	decoder.Decode(&searchResponse)

	if len(searchResponse) <= 0 {
		t.Fatalf("Data Error. %v", searchResponse)
	}
}

func TestQuery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(query))
	defer ts.Close()
	client := &http.Client{}

	// Test for an OPTIONS request
	reqOptions, err := http.NewRequest("OPTIONS", ts.URL, nil)
	if err != nil {
		t.Fatalf("Error creating an OPTIONS request. %v", err)
	}
	r, err := client.Do(reqOptions)
	if err != nil {
		t.Fatalf("Error at an OPTIONS request. %v", err)
	}

	if r.StatusCode != 200 {
		t.Fatalf("Status code is not 200 but %d.", r.StatusCode)
	}

	if r.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("Header Access-Control-Allow-Origin is invalid. %s", r.Header.Get("Access-Control-Allow-Origin"))
	}
	if r.Header.Get("Access-Control-Allow-Headers") != "accept, content-type" {
		t.Fatalf("Header Access-Control-Allow-Headers is invalid. %s", r.Header.Get("Access-Control-Allow-Headers"))
	}
	if r.Header.Get("Access-Control-Allow-Methods") != "GET,POST,HEAD,OPTIONS" {
		t.Fatalf("Header Access-Control-Allow-Methods is invalid. %s", r.Header.Get("Access-Control-Allow-Methods"))
	}

	// Test for a POST request
	requestJSON := `{
	  "panelId":1,
	  "range":{
	    "from":"2017-01-17T05:14:42.237Z",
	    "to":"2017-01-18T05:14:42.237Z",
	    "raw":{
	      "from":"now-24h",
	      "to":"now"
	    }
	  },
	  "rangeRaw":{
	    "from":"now-24h",
	    "to":"now"
	  },
	  "interval":"1m",
	  "intervalMs":60000,
	  "targets":[
	    {"target":"sample:ClientJobsIdle","refId":"A","hide":false,"type":"timeserie"},
	    {"target":"sample:ClientJobsRunning","refId":"B","hide":false,"type":"timeserie"},
			{"target":"percent-*:value","refId":"B","hide":false,"type":"timeserie"}
	  ],
	  "format":"json",
	  "maxDataPoints":1812
	}`

	reader := strings.NewReader(requestJSON)
	r, err = http.Post(ts.URL, "application/json; charset=utf-8", reader)
	if err != nil {
		t.Fatalf("Error at an POST request. %v", err)
	}

	if r.StatusCode != 200 {
		t.Fatalf("Status code is not 200 but %d.", r.StatusCode)
	}

	if r.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("Header Access-Control-Allow-Origin is invalid. %s", r.Header.Get("Access-Control-Allow-Origin"))
	}
	if r.Header.Get("Access-Control-Allow-Headers") != "accept, content-type" {
		t.Fatalf("Header Access-Control-Allow-Headers is invalid. %s", r.Header.Get("Access-Control-Allow-Headers"))
	}
	if r.Header.Get("Access-Control-Allow-Methods") != "GET,POST,HEAD,OPTIONS" {
		t.Fatalf("Header Access-Control-Allow-Methods is invalid. %s", r.Header.Get("Access-Control-Allow-Methods"))
	}

	decoder := json.NewDecoder(r.Body)
	var qrs = []QueryResponse{}
	err = decoder.Decode(&qrs)
	if err != nil {
		t.Fatalf("Error at decoding JSON response. %v", err)
	}

	clientJobsIdleExists := false
	percentIdleExists := false
	percentUserExists := false
	for _, v := range qrs {
		if len(v.Target) < 0 {
			t.Fatalf("Response is empty.")
		}
		if v.Target == "sample:ClientJobsIdle" {
			clientJobsIdleExists = true
		}
		if v.Target == "sample:percent-idle:value" {
			percentIdleExists = true
		}
		if v.Target == "sample:percent-user:value" {
			percentUserExists = true
		}
	}
	if !clientJobsIdleExists {
		t.Fatal("sample:ClientJobsIdle isn't contained in the response.")
	}
	if !percentIdleExists {
		t.Fatal("sample:percent-idle:value isn't contained in the response.")
	}
	if !percentUserExists {
		t.Fatal("sample:percent-user:value isn't contained in the response.")
	}
}

func TestAnnotations(t *testing.T) {
	config.Server.AnnotationFilePath = "./sample/annotations.csv"

	ts := httptest.NewServer(http.HandlerFunc(annotations))
	defer ts.Close()

	requestJSON := `{
		"range": {
			"from": "2017-05-14T00:00:00.000Z",
			"to": "2017-05-14T23:59:59.000Z"
		},
		"rangeRaw": {
			"from": "now-24h",
			"to": "now"
		},
		"annotation": {
			"name": "deploy",
			"datasource": "Simple JSON Datasource",
			"iconColor": "rgba(255, 96, 96, 1)",
			"enable": true,
			"query": "#deploy"
		}
	}`

	reader := strings.NewReader(requestJSON)
	r, err := http.Post(ts.URL, "application/json; charset=utf-8", reader)
	if err != nil {
		t.Fatalf("Error at an POST request. %v", err)
	}

	if r.StatusCode != 200 {
		t.Fatalf("Status code is not 200 but %d.", r.StatusCode)
		return
	}

	if r.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("Header Access-Control-Allow-Origin is invalid. %s", r.Header.Get("Access-Control-Allow-Origin"))
	}
	if r.Header.Get("Access-Control-Allow-Headers") != "accept, content-type" {
		t.Fatalf("Header Access-Control-Allow-Headers is invalid. %s", r.Header.Get("Access-Control-Allow-Headers"))
	}
	if r.Header.Get("Access-Control-Allow-Methods") != "GET,POST,HEAD,OPTIONS" {
		t.Fatalf("Header Access-Control-Allow-Methods is invalid. %s", r.Header.Get("Access-Control-Allow-Methods"))
	}

	decoder := json.NewDecoder(r.Body)
	var ars = []AnnotationResponse{}
	err = decoder.Decode(&ars)
	if err != nil {
		t.Fatalf("Error at decoding JSON response. %v", err)
	}

	timeExists := false
	titleExists := false
	timeOutSideRangeExists := false
	titleOutSideRangeExists := false
	for _, v := range ars {
		if len(v.Title) < 0 {
			t.Fatalf("Response is empty.")
		}
		if v.Time == 1494763899000 {
			timeExists = true
		}
		if v.Title == "App restarted" {
			titleExists = true
		}

		if v.Time == 1480917950000 {
			timeOutSideRangeExists = true
		}
		if v.Title == "Rebooted" {
			titleOutSideRangeExists = true
		}
	}

	if !timeExists {
		t.Fatal("The expected time value isn't contained in the response.")
	}
	if !titleExists {
		t.Fatal("The expected time value isn't contained in the response.")
	}
	if timeOutSideRangeExists {
		t.Fatal("The time value that shouldn't be contained is found.")
	}
	if titleOutSideRangeExists {
		t.Fatal("The title value that shouldn't be contained is found.")
	}
}

func TestSetArgs(t *testing.T) {

}
