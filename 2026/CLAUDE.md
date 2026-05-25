# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Context

Rinha de Backend 2026 — a competition to build the fastest + most accurate fraud-detection API. The score combines **latency** (p99, up to +3000) and **detection quality** (false positives/negatives/HTTP errors, up to +3000). The critical threshold: if more than 15% of requests fail detection, the detection score is cut to −3000 regardless of latency.

The API vectorizes incoming JSON transactions into 14-dimensional float32 vectors and performs approximate KNN (k=5) against a 3M-record reference dataset. `fraud_score = frauds_among_5_neighbors / 5`; approved if `fraud_score < 0.6`.

## Commands

```bash
# Start the full stack (HAProxy + 2 API instances)
docker compose up --build -d
docker compose logs -f          # wait for "Engine loaded with 3000000 records"

# Run tests
k6 run test/smoke.js            # 5 quick requests
k6 run test/test.js             # official test → writes test/results.json

# Stress benchmark (requires local Go install)
go run benchmark.go -n 100 -c 4
go run benchmark.go -n 1000 -c 20

# Rebuild the binary index only (runs inside Docker during build)
go run ./cmd/converter/main.go -in resources/references.json.gz -out references.bin

# Teardown
docker compose down
```

Infrastructure limits: HAProxy ≤ 0.10 CPU / 15 MB, each API ≤ 0.45 CPU / 165 MB, total ≤ 1 CPU / 350 MB.

## Architecture

### Build pipeline (two-stage Docker build)

1. **`cmd/converter/main.go`** — build-time only. Reads `resources/references.json.gz` (3M pre-vectorized records, each `{is_fraud, vector[14]}`), runs K-Means clustering (1024 clusters, 15 iterations, 100k sample), and writes a binary IVF index (`references.bin`) in the `IVF1` format.

2. **`cmd/api/main.go`** — runtime. Loads `references.bin` via mmap at startup, then serves:
   - `GET /ready` — liveness check (responds once data is loaded)
   - `POST /fraud-score` — vectorizes the JSON payload and returns `{approved, fraud_score}`

### Engine (`internal/engine/`)

| File | Role |
|------|------|
| `data.go` | `DataEngine` struct: mmaps `references.bin`, exposes `Clusters []Cluster` and `Records []VectorRecord` |
| `vectorizer.go` | `Vectorize()` — converts raw JSON body to `[14]float32` using hardcoded normalization constants (match `resources/normalization.json`) |
| `search.go` | `SearchNeighbors()` — IVF approximate KNN: finds top `NPROBE` clusters via `batchDistAVX2_64`, then scans each cluster via `batchDistAVX2` |
| `simd_amd64.s` | AVX2 assembly kernels: `batchDistAVX2` (60-byte record stride) and `batchDistAVX2_64` (64-byte cluster-centroid stride) |
| `simd_generic.go` | Pure-Go fallback for non-amd64 (used in Mac M1/M2 dev) |

### Binary index format (`IVF1`)

```
[4 bytes magic "IVF1"]
[4 bytes numClusters]
[numClusters × 64 bytes: centroid[14]float32 + Start uint32 + Count uint32]
[N × 60 bytes: Dimensions[14]float32 + IsFraud uint8 + 3 bytes padding]
```

Records are sorted by cluster assignment (Start/Count point into the record array).

### Key constants to tune

| Constant | Location | Current | Notes |
|----------|----------|---------|-------|
| `NPROBE` | `search.go:19` | 32 | Clusters probed per query (higher = more accurate, slower) |
| `numClusters` | `converter/main.go:42` | 1024 | IVF clusters (more = less records/cluster = faster per probe) |
| `subsetSize` | `converter/main.go:102` | 100k | K-Means training sample (larger = better centroids, slower build) |
| KMeans iterations | `converter/main.go:115` | 15 | More = better convergence |

### Scoring formula (from `test/test.js`)

- **p99 score**: `1000 × log10(1000 / max(p99_ms, 1))`, capped at ±3000 (cut if p99 > 2000ms)
- **Detection score**: `1000 × log10(1/ε) − 300 × log10(1 + E)` where `E = FP×1 + FN×3 + HTTP_err×5` and `ε = E/N`, cut to −3000 if failure_rate > 15%

**Important**: returning HTTP 500 costs weight 5 vs FN weight 3. On any error, return `{"approved":true,"fraud_score":0}` instead.
