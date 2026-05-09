package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"rinha_de_backend_2026/internal/engine"
)

var mccRisks = map[string]float32{
	"5411": 0.15,
	"5812": 0.30,
	"5912": 0.20,
	"5944": 0.45,
	"7801": 0.80,
	"7802": 0.75,
	"7995": 0.85,
	"4511": 0.35,
	"5311": 0.25,
	"5999": 0.50,
}

var dataEngine *engine.DataEngine

func main() {
	var err error
	dataEngine, err = engine.LoadEngine("/data/references.bin")
	if err != nil {
		log.Printf("Failed to load /data/references.bin: %v", err)
		log.Printf("Attempting to load ./references.bin locally...")
		dataEngine, err = engine.LoadEngine("references.bin")
		if err != nil {
			log.Fatalf("Fatal: could not load references.bin: %v", err)
		}
	}
	defer dataEngine.Close()

	log.Printf("Engine loaded with %d records", len(dataEngine.Records))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ready", handleReady)
	mux.HandleFunc("POST /fraud-score", handleFraudScore)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleFraudScore(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "bad request"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	vector, err := engine.Vectorize(body, mccRisks)
	if err != nil {
		http.Error(w, `{"error": "invalid payload"}`, http.StatusBadRequest)
		return
	}

	score := engine.SearchNeighbors(vector, dataEngine)
	approved := score < 0.6

	response := struct {
		Approved   bool    `json:"approved"`
		FraudScore float32 `json:"fraud_score"`
	}{
		Approved:   approved,
		FraudScore: score,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
