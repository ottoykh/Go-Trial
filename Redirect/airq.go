package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type Coordinates struct {
	Longitude float64
	Latitude  float64
}

var coordinates = map[string]Coordinates{
	"Southern":        {114.16014, 22.247461},
	"North":           {114.128244, 22.496697},
	"Kwun Tong":       {114.231174, 22.309625},
	"Tseung Kwan O":   {114.259561, 22.317642},
	"Tuen Mun":        {113.976728, 22.391143},
	"Tung Chung":      {113.943659, 22.288889},
	"Eastern Air":     {114.219372, 22.282886},
	"Tap Mun":         {114.360719, 22.471317},
	"Kwai Chung":      {114.129601, 22.357104},
	"Yuen Long":       {114.022649, 22.445155},
	"Sha Tin":         {114.184532, 22.376281},
	"Sham Shui Po":    {114.159109, 22.330226},
	"Tai Po":          {114.16457, 22.45096},
	"Mong Kok":        {114.168272, 22.322611},
	"Central/Western": {114.144421, 22.284891},
	"Central":         {114.158127, 22.281815},
	"Causeway Bay":    {114.18509, 22.280133},
	"Tsuen Wan":       {114.114535, 22.371742},
}

func getCachedData(key string, ttl int) ([]byte, bool) {
	cacheFile := filepath.Join(os.TempDir(), fmt.Sprintf("aqhi_cache_%x", key))
	info, err := os.Stat(cacheFile)
	if err == nil && time.Since(info.ModTime()) < time.Duration(ttl)*time.Second {
		data, err := ioutil.ReadFile(cacheFile)
		if err == nil {
			return data, true
		}
	}
	return nil, false
}

func setCachedData(key string, data []byte) {
	cacheFile := filepath.Join(os.TempDir(), fmt.Sprintf("aqhi_cache_%x", key))
	_ = ioutil.WriteFile(cacheFile, data, 0644)
}

func fetchAndExtractJSON(url string, variableName string) ([]interface{}, error) {
	cacheKey := url + variableName
	if data, ok := getCachedData(cacheKey, 300); ok {
		var result []interface{}
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(fmt.Sprintf(`var %s = (\[.+?\]);`, regexp.QuoteMeta(variableName)))
	match := re.FindSubmatch(body)
	if len(match) < 2 {
		log.Printf("Failed to find variable %s in the response body.\nResponse Body: %s\n", variableName, body)
		return nil, fmt.Errorf("variable not found")
	}

	var result []interface{}
	if err := json.Unmarshal(match[1], &result); err != nil {
		log.Printf("Failed to unmarshal JSON for variable %s.\nJSON: %s\nError: %s\n", variableName, match[1], err)
		return nil, err
	}

	setCachedData(cacheKey, match[1])
	return result, nil
}

func getData(last, recent bool) (map[string]interface{}, error) {
	url := "https://www.aqhi.gov.hk/js/data/past_24_pollutant.js"
	data, err := fetchAndExtractJSON(url, "station_24_data")
	if err != nil {
		return nil, err
	}

	features := make(map[string]interface{})
	for _, stationData := range data {
		for _, entry := range stationData.([]interface{}) {
			entryMap := entry.(map[string]interface{})
			stationName := entryMap["StationNameEN"].(string)
			if coords, ok := coordinates[stationName]; ok {
				measurement := map[string]interface{}{
					"DateTime": entryMap["DateTime"],
					"aqhi":     entryMap["aqhi"],
					"NO2":      entryMap["NO2"],
					"O3":       entryMap["O3"],
					"SO2":      entryMap["SO2"],
					"CO":       entryMap["CO"],
					"PM10":     entryMap["PM10"],
					"PM25":     entryMap["PM25"],
				}

				feature, found := features[stationName]
				if !found {
					feature = map[string]interface{}{
						"type": "Feature",
						"geometry": map[string]interface{}{
							"type":        "Point",
							"coordinates": []float64{coords.Longitude, coords.Latitude},
						},
						"properties": map[string]interface{}{
							"name":    stationName,
							"feature": []map[string]interface{}{measurement},
						},
					}
				} else {
					feature.(map[string]interface{})["properties"].(map[string]interface{})["feature"] = append(
						feature.(map[string]interface{})["properties"].(map[string]interface{})["feature"].([]map[string]interface{}),
						measurement,
					)
				}
				features[stationName] = feature
			}
		}
	}

	result := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
	}

	if last || recent {
		for _, feature := range features {
			featureMap := feature.(map[string]interface{})
			if features, ok := featureMap["properties"].(map[string]interface{})["feature"].([]map[string]interface{}); ok && len(features) > 0 {
				if last {
					featureMap["properties"].(map[string]interface{})["feature"] = features[len(features)-1 : len(features)]
				} else if recent {
					featureMap["properties"].(map[string]interface{})["feature"] = features[0:1]
				}
			}
		}
	}

	return result, nil
}

func getAQHIReportAndForecast(w http.ResponseWriter, r *http.Request) {
	url := "https://www.aqhi.gov.hk/js/data/forecast_aqhi.js"
	aqhiReport, err := fetchAndExtractJSON(url, "aqhi_report")
	responseData := make(map[string]interface{})

	if err != nil {
		responseData["aqhi_report"] = map[string]string{"error": "No match found for aqhi_report."}
	} else {
		responseData["aqhi_report"] = aqhiReport
	}

	aqhiForecast, err := fetchAndExtractJSON(url, "aqhi_forecast")
	if err != nil {
		responseData["aqhi_forecast"] = map[string]string{"error": "No match found for aqhi_forecast."}
	} else {
		responseData["aqhi_forecast"] = aqhiForecast
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dataType := r.URL.Query().Get("data_type")
	last, _ := strconv.ParseBool(r.URL.Query().Get("last"))
	recent, _ := strconv.ParseBool(r.URL.Query().Get("recent"))

	var result map[string]interface{}
	var err error

	switch dataType {
	case "data":
		result, err = getData(last, recent)
	case "repo":
		getAQHIReportAndForecast(w, r)
		return
	default:
		result = map[string]interface{}{"error": "Invalid data_type."}
	}

	if err != nil {
		result = map[string]interface{}{"error": err.Error()}
	}

	json.NewEncoder(w).Encode(result)
}

func main() {
	http.HandleFunc("/", handleRequest)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
