package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

// Normalization mirrors the normalization.json shape.
type Normalization struct {
	MaxAmount            float32 `json:"max_amount"`
	MaxInstallments      float32 `json:"max_installments"`
	AmountVsAvgRatio     float32 `json:"amount_vs_avg_ratio"`
	MaxMinutes           float32 `json:"max_minutes"`
	MaxKm                float32 `json:"max_km"`
	MaxTxCount24h        float32 `json:"max_tx_count_24h"`
	MaxMerchantAvgAmount float32 `json:"max_merchant_avg_amount"`
}

// ReferenceRecord is the expected shape of each JSON entry in references.json.gz.
type ReferenceRecord struct {
	IsFraud bool        `json:"is_fraud"`
	Vector  [14]float32 `json:"vector"`
}

type VectorRecord struct {
	Vector  [14]float32
	IsFraud uint8
}

const numClusters = 1024

func readJSONFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func run() error {
	inputPath := flag.String("in", "resources/references.json.gz", "path to references.json.gz")
	outputPath := flag.String("out", "references.bin", "output path")
	flag.Parse()

	start := time.Now()

	inputFile, err := os.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer inputFile.Close()

	gzReader, err := gzip.NewReader(inputFile)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer gzReader.Close()

	decoder := json.NewDecoder(gzReader)
	startToken, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("read json start token: %w", err)
	}
	if startDelim, ok := startToken.(json.Delim); !ok || startDelim != '[' {
		return fmt.Errorf("expected JSON array start, got %v", startToken)
	}

	var records []VectorRecord
	count := 0
	for decoder.More() {
		var rec ReferenceRecord
		if err := decoder.Decode(&rec); err != nil {
			return fmt.Errorf("decode record %d: %w", count, err)
		}
		vr := VectorRecord{Vector: rec.Vector}
		if rec.IsFraud {
			vr.IsFraud = 1
		}
		records = append(records, vr)
		count++
		if count%500000 == 0 {
			log.Printf("Loaded %d records", count)
		}
	}

	log.Printf("Loaded %d records in %v. Starting KMeans with %d clusters...", len(records), time.Since(start), numClusters)

	// Training subset
	subsetSize := 100000
	if len(records) < subsetSize {
		subsetSize = len(records)
	}
	r := rand.New(rand.NewSource(42))
	subsetIndices := r.Perm(len(records))[:subsetSize]

	// Init centroids
	var centroids [numClusters][14]float32
	for i := 0; i < numClusters; i++ {
		centroids[i] = records[subsetIndices[i]].Vector
	}

	// Train KMeans
	for iter := 0; iter < 15; iter++ {
		var sums [numClusters][14]float32
		var counts [numClusters]int

		for _, idx := range subsetIndices {
			vec := records[idx].Vector
			bestC, bestDist := 0, float32(math.MaxFloat32)
			for c := 0; c < numClusters; c++ {
				d := distSq(vec, centroids[c])
				if d < bestDist {
					bestDist = d
					bestC = c
				}
			}
			counts[bestC]++
			for i := 0; i < 14; i++ {
				sums[bestC][i] += vec[i]
			}
		}

		changed := 0
		for c := 0; c < numClusters; c++ {
			if counts[c] > 0 {
				for i := 0; i < 14; i++ {
					newVal := sums[c][i] / float32(counts[c])
					if centroids[c][i] != newVal {
						centroids[c][i] = newVal
						changed++
					}
				}
			}
		}
		log.Printf("KMeans iter %d done, changed %d dimensions", iter+1, changed)
	}

	log.Println("Assigning all records to clusters (parallel)...")
	assignments := make([]uint32, len(records))
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	if workers < 4 {
		workers = 4
	}
	chunk := len(records) / workers
	if chunk == 0 {
		chunk = 1
	}

	for i := 0; i < workers; i++ {
		startIdx := i * chunk
		endIdx := startIdx + chunk
		if i == workers-1 {
			endIdx = len(records)
		}
		if startIdx >= len(records) {
			break
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for j := s; j < e; j++ {
				vec := records[j].Vector
				bestC, bestDist := uint32(0), float32(math.MaxFloat32)
				for c := 0; c < numClusters; c++ {
					d := distSq(vec, centroids[c])
					if d < bestDist {
						bestDist = d
						bestC = uint32(c)
					}
				}
				assignments[j] = bestC
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	log.Println("Sorting records by cluster...")
	indices := make([]uint32, len(records))
	for i := range indices {
		indices[i] = uint32(i)
	}
	// Sort primarily by assignment
	sort.Slice(indices, func(i, j int) bool {
		return assignments[indices[i]] < assignments[indices[j]]
	})

	log.Println("Writing IVF binary format to disk...")
	outputFile, err := os.Create(*outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outputFile.Close()
	writer := bufio.NewWriterSize(outputFile, 4<<20)

	// Write Magic
	writer.Write([]byte("IVF1"))

	// Write NumClusters
	var buf4 [4]byte
	binary.LittleEndian.PutUint32(buf4[:], numClusters)
	writer.Write(buf4[:])

	var clusterStarts [numClusters]uint32
	var clusterCounts [numClusters]uint32
	for _, idx := range indices {
		c := assignments[idx]
		clusterCounts[c]++
	}

	startAcc := uint32(0)
	for c := 0; c < numClusters; c++ {
		clusterStarts[c] = startAcc
		startAcc += clusterCounts[c]
	}

	// Write Centroids Header (64 bytes each)
	var hbuf [64]byte
	for c := 0; c < numClusters; c++ {
		for i := 0; i < 14; i++ {
			binary.LittleEndian.PutUint32(hbuf[i*4:], math.Float32bits(centroids[c][i]))
		}
		binary.LittleEndian.PutUint32(hbuf[56:], clusterStarts[c])
		binary.LittleEndian.PutUint32(hbuf[60:], clusterCounts[c])
		writer.Write(hbuf[:])
	}

	// Write Records
	var rbuf [60]byte
	for _, idx := range indices {
		rec := records[idx]
		for i := 0; i < 14; i++ {
			binary.LittleEndian.PutUint32(rbuf[i*4:], math.Float32bits(rec.Vector[i]))
		}
		rbuf[56] = rec.IsFraud
		rbuf[57], rbuf[58], rbuf[59] = 0, 0, 0
		writer.Write(rbuf[:])
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	log.Printf("Converter done in %v! output=%s", time.Since(start), *outputPath)
	return nil
}

func distSq(a, b [14]float32) float32 {
	d0 := a[0] - b[0]
	d1 := a[1] - b[1]
	d2 := a[2] - b[2]
	d3 := a[3] - b[3]
	d4 := a[4] - b[4]
	d5 := a[5] - b[5]
	d6 := a[6] - b[6]
	d7 := a[7] - b[7]
	d8 := a[8] - b[8]
	d9 := a[9] - b[9]
	d10 := a[10] - b[10]
	d11 := a[11] - b[11]
	d12 := a[12] - b[12]
	d13 := a[13] - b[13]
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5 + d6*d6 + d7*d7 + d8*d8 + d9*d9 + d10*d10 + d11*d11 + d12*d12 + d13*d13
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
