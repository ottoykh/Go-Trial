package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type GeoJSONFeature struct {
	Type       string            `json:"type"`
	Geometry   GeoJSONGeometry   `json:"geometry"`
	Properties GeoJSONProperties `json:"properties"`
}

type GeoJSONGeometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

type GeoJSONProperties struct {
	Station        string  `json:"Automatic Weather Station"`
	AirTemperature float64 `json:"Air Temperature"`
}

type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

var features []GeoJSONFeature

func main() {
	features = []GeoJSONFeature{
		{
			Type: "Feature",
			Geometry: GeoJSONGeometry{
				Type:        "Point",
				Coordinates: [2]float64{113.9219444, 22.3094444},
			},
			Properties: GeoJSONProperties{
				Station:        "Chek Lap Kok",
				AirTemperature: 27.3,
			},
		},
	}

	router := mux.NewRouter()

	router.HandleFunc("/api/features", getFeatures).Methods("GET")
	router.HandleFunc("/api/features/{id:[0-9]+}", getFeature).Methods("GET")
	router.HandleFunc("/api/features", createFeature).Methods("POST")
	router.HandleFunc("/api/features/{id:[0-9]+}", updateFeature).Methods("PUT")
	router.HandleFunc("/api/features/{id:[0-9]+}", deleteFeature).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":1234", router))
}

func getFeatures(w http.ResponseWriter, r *http.Request) {
	collection := GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

func getFeature(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if id < 0 || id >= len(features) {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(features[id])
}

func createFeature(w http.ResponseWriter, r *http.Request) {
	var feature GeoJSONFeature
	err := json.NewDecoder(r.Body).Decode(&feature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	feature.Geometry = GeoJSONGeometry{
		Type:        "Point",
		Coordinates: [2]float64{113, 22},
	}

	features = append(features, feature)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feature)
}

func updateFeature(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if id < 0 || id >= len(features) {
		http.NotFound(w, r)
		return
	}

	var updatedFeature GeoJSONFeature
	err = json.NewDecoder(r.Body).Decode(&updatedFeature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updatedFeature.Geometry = features[id].Geometry

	features[id] = updatedFeature

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedFeature)
}

func deleteFeature(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if id < 0 || id >= len(features) {
		http.NotFound(w, r)
		return
	}

	features = append(features[:id], features[id+1:]...)

	w.WriteHeader(http.StatusNoContent)
}
