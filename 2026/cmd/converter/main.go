package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
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

func readJSONFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return err
	}

	return nil
}

func run() error {
	inputPath := flag.String("in", "resources/references.json.gz", "path to references.json.gz")
	normPath := flag.String("norm", "resources/normalization.json", "path to normalization.json")
	mccPath := flag.String("mcc", "resources/mcc_risk.json", "path to mcc_risk.json")
	outputPath := flag.String("out", "references.bin", "output path")
	logEvery := flag.Int("log-every", 100000, "log progress every N records (0 disables)")
	bufferSize := flag.Int("buffer", 4<<20, "output buffer size in bytes")
	flag.Parse()

	if *bufferSize < 60 {
		return fmt.Errorf("buffer size %d is too small", *bufferSize)
	}

	start := time.Now()
	log.Printf("converter: loading normalization from %s", *normPath)

	var norm Normalization
	if err := readJSONFile(*normPath, &norm); err != nil {
		return fmt.Errorf("read normalization: %w", err)
	}

	log.Printf("converter: loading mcc risk from %s", *mccPath)
	var mccRisk map[string]float32
	if err := readJSONFile(*mccPath, &mccRisk); err != nil {
		return fmt.Errorf("read mcc risk: %w", err)
	}

	log.Printf("converter: loaded normalization and %d mcc entries", len(mccRisk))

	inputFile, err := os.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer func() {
		if cerr := inputFile.Close(); cerr != nil {
			log.Printf("converter: input close warning: %v", cerr)
		}
	}()

	gzReader, err := gzip.NewReader(inputFile)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer func() {
		if cerr := gzReader.Close(); cerr != nil {
			log.Printf("converter: gzip close warning: %v", cerr)
		}
	}()

	outputFile, err := os.Create(*outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer func() {
		if cerr := outputFile.Close(); cerr != nil {
			log.Printf("converter: output close warning: %v", cerr)
		}
	}()

	writer := bufio.NewWriterSize(outputFile, *bufferSize)
	decoder := json.NewDecoder(gzReader)

	startToken, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("read json start token: %w", err)
	}
	startDelim, ok := startToken.(json.Delim)
	if !ok || startDelim != '[' {
		return fmt.Errorf("expected JSON array start, got %v", startToken)
	}

	count := 0
	var buf [60]byte

	for decoder.More() {
		var record ReferenceRecord
		if err := decoder.Decode(&record); err != nil {
			return fmt.Errorf("decode record %d: %w", count, err)
		}

		// Fixed 60-byte layout: 14 float32 (56 bytes) + 1 byte flag + 3 bytes padding.
		for i := 0; i < 14; i++ {
			binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(record.Vector[i]))
		}

		if record.IsFraud {
			buf[56] = 1
		} else {
			buf[56] = 0
		}
		buf[57], buf[58], buf[59] = 0, 0, 0

		if _, err := writer.Write(buf[:]); err != nil {
			return fmt.Errorf("write record %d: %w", count, err)
		}

		count++
		if *logEvery > 0 && count%*logEvery == 0 {
			log.Printf("converter: processed %d records", count)
		}
	}

	endToken, err := decoder.Token()
	if err != nil && err != io.EOF {
		return fmt.Errorf("read json end token: %w", err)
	}
	if err == nil {
		endDelim, ok := endToken.(json.Delim)
		if !ok || endDelim != ']' {
			return fmt.Errorf("expected JSON array end, got %v", endToken)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	elapsed := time.Since(start)
	log.Printf("converter: done. records=%d elapsed=%s output=%s", count, elapsed, *outputPath)

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
