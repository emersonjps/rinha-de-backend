package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

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

var bodyPool = sync.Pool{
	New: func() any {
		b := make([]byte, 8192)
		return &b
	},
}

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
	bptr := bodyPool.Get().(*[]byte)
	buf := *bptr

	var n int
	for {
		c, err := r.Body.Read(buf[n:])
		n += c
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, `{"error": "bad request"}`, http.StatusBadRequest)
			bodyPool.Put(bptr)
			return
		}
		if n == len(buf) {
			break
		}
	}
	r.Body.Close()

	vector, err := engine.Vectorize(buf[:n], mccRisks)
	if err != nil {
		http.Error(w, `{"error": "invalid payload"}`, http.StatusBadRequest)
		bodyPool.Put(bptr)
		return
	}

	score := engine.SearchNeighbors(vector, dataEngine)
	approved := score < 0.6

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"approved":%t,"fraud_score":%f}`, approved, score)

	bodyPool.Put(bptr)
}
