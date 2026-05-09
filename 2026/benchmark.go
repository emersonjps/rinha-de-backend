package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

func main() {
	concurrency := flag.Int("c", 10, "concurrency")
	requests := flag.Int("n", 1000, "number of requests")
	url := flag.String("u", "http://localhost:9999/fraud-score", "URL")
	flag.Parse()

	// Load payloads
	data, err := os.ReadFile("resources/example-payloads.json")
	if err != nil {
		panic(err)
	}
	var payloads []map[string]interface{}
	if err := json.Unmarshal(data, &payloads); err != nil {
		panic(err)
	}
	if len(payloads) == 0 {
		panic("no payloads")
	}

	payloadBytes := make([][]byte, len(payloads))
	for i, p := range payloads {
		b, _ := json.Marshal(p)
		payloadBytes[i] = b
	}

	fmt.Printf("Running %d requests with concurrency %d...\n", *requests, *concurrency)

	var wg sync.WaitGroup
	reqChan := make(chan []byte, *requests)
	for i := 0; i < *requests; i++ {
		reqChan <- payloadBytes[i%len(payloadBytes)]
	}
	close(reqChan)

	var mu sync.Mutex
	var latencies []time.Duration
	success := 0
	failed := 0

	start := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Transport: &http.Transport{
					MaxIdleConnsPerHost: *concurrency,
					IdleConnTimeout:     30 * time.Second,
				},
			}
			for p := range reqChan {
				reqStart := time.Now()
				req, _ := http.NewRequest("POST", *url, bytes.NewReader(p))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				
				latency := time.Since(reqStart)
				
				mu.Lock()
				if resp.StatusCode == 200 {
					success++
				} else {
					failed++
				}
				latencies = append(latencies, latency)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	total := time.Since(start)

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	fmt.Println("\n--- Benchmark Results ---")
	fmt.Printf("Total Time:  %v\n", total)
	fmt.Printf("Throughput:  %.2f req/s\n", float64(success)/total.Seconds())
	fmt.Printf("Success:     %d\n", success)
	fmt.Printf("Failed:      %d\n", failed)
	
	if len(latencies) > 0 {
		fmt.Printf("\n--- Latency Distribution ---\n")
		fmt.Printf("P50:         %v\n", latencies[len(latencies)*50/100])
		fmt.Printf("P90:         %v\n", latencies[len(latencies)*90/100])
		fmt.Printf("P95:         %v\n", latencies[len(latencies)*95/100])
		fmt.Printf("P99:         %v\n", latencies[len(latencies)*99/100])
		fmt.Printf("Max Latency: %v\n", latencies[len(latencies)-1])
	}
}
