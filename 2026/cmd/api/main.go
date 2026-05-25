package main

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"rinha_de_backend_2026/internal/engine"
)

var mccRiskArray [10000]float32

func init() {
	for i := 0; i < 10000; i++ {
		mccRiskArray[i] = 0.5
	}
	mccRiskArray[5411] = 0.15
	mccRiskArray[5812] = 0.30
	mccRiskArray[5912] = 0.20
	mccRiskArray[5944] = 0.45
	mccRiskArray[7801] = 0.80
	mccRiskArray[7802] = 0.75
	mccRiskArray[7995] = 0.85
	mccRiskArray[4511] = 0.35
	mccRiskArray[5311] = 0.25
	mccRiskArray[5999] = 0.50
}

var dataEngine *engine.DataEngine

var bodyPool = sync.Pool{
	New: func() any {
		b := make([]byte, 8192)
		return &b
	},
}

var respPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 128)
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"approved":true,"fraud_score":0.0}`))
			bodyPool.Put(bptr)
			return
		}
		if n == len(buf) {
			break
		}
	}
	r.Body.Close()

	vector, err := engine.Vectorize(buf[:n], &mccRiskArray)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"approved":true,"fraud_score":0.0}`))
		bodyPool.Put(bptr)
		return
	}

	score := engine.SearchNeighbors(vector, dataEngine)
	approved := score < 0.6

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := respPool.Get().([]byte)
	resp = resp[:0]
	resp = append(resp, `{"approved":`...)
	if approved {
		resp = append(resp, "true"...)
	} else {
		resp = append(resp, "false"...)
	}
	resp = append(resp, `,"fraud_score":`...)
	resp = strconv.AppendFloat(resp, float64(score), 'f', 4, 32)
	resp = append(resp, '}')
	
	w.Write(resp)
	
	respPool.Put(resp)
	bodyPool.Put(bptr)
}
