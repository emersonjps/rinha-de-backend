package engine

import (
	"math"
	"runtime"
	"sync"
)

const maxWorkers = 2

type Neighbor struct {
	DistSq  float32
	IsFraud uint8
}

// SearchNeighbors performs a brute-force KNN (k=5) using a linear scan.
func SearchNeighbors(target [14]float32, records []VectorRecord) float32 {
	if len(records) == 0 {
		return 0
	}

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > maxWorkers {
		workers = maxWorkers
	}
	if workers > len(records) {
		workers = 1
	}

	if workers == 1 {
		best := searchChunk(target, records)
		return fraudScore(best)
	}

	chunkSize := len(records) / workers
	if chunkSize == 0 {
		best := searchChunk(target, records)
		return fraudScore(best)
	}

	var wg sync.WaitGroup
	var locals [maxWorkers][5]Neighbor

	for i := 0; i < workers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == workers-1 {
			end = len(records)
		}

		wg.Add(1)
		go func(workerID, startIdx, endIdx int) {
			defer wg.Done()
			heap := searchChunk(target, records[startIdx:endIdx])
			locals[workerID] = heap
		}(i, start, end)
	}

	wg.Wait()

	best := initHeap()
	for i := 0; i < workers; i++ {
		for j := 0; j < 5; j++ {
			insertNeighbor(&best, locals[i][j])
		}
	}

	return fraudScore(best)
}

func searchChunk(target [14]float32, records []VectorRecord) [5]Neighbor {
	heap := initHeap()

	for i := range records {
		rec := &records[i]

		d0 := rec.Dimensions[0] - target[0]
		d1 := rec.Dimensions[1] - target[1]
		d2 := rec.Dimensions[2] - target[2]
		d3 := rec.Dimensions[3] - target[3]
		d4 := rec.Dimensions[4] - target[4]
		d5 := rec.Dimensions[5] - target[5]
		d6 := rec.Dimensions[6] - target[6]
		d7 := rec.Dimensions[7] - target[7]
		d8 := rec.Dimensions[8] - target[8]
		d9 := rec.Dimensions[9] - target[9]
		d10 := rec.Dimensions[10] - target[10]
		d11 := rec.Dimensions[11] - target[11]
		d12 := rec.Dimensions[12] - target[12]
		d13 := rec.Dimensions[13] - target[13]

		distSq := d0*d0 + d1*d1 + d2*d2 + d3*d3 +
			d4*d4 + d5*d5 + d6*d6 + d7*d7 +
			d8*d8 + d9*d9 + d10*d10 + d11*d11 +
			d12*d12 + d13*d13

		if distSq < heap[4].DistSq {
			heap[4] = Neighbor{DistSq: distSq, IsFraud: rec.IsFraud}
			bubbleUp(&heap)
		}
	}

	return heap
}

func initHeap() [5]Neighbor {
	var heap [5]Neighbor
	maxDist := float32(math.MaxFloat32)
	for i := 0; i < 5; i++ {
		heap[i].DistSq = maxDist
	}
	return heap
}

func insertNeighbor(heap *[5]Neighbor, n Neighbor) {
	if n.DistSq >= heap[4].DistSq {
		return
	}
	heap[4] = n
	bubbleUp(heap)
}

func bubbleUp(heap *[5]Neighbor) {
	for i := 4; i > 0; i-- {
		if heap[i].DistSq < heap[i-1].DistSq {
			heap[i], heap[i-1] = heap[i-1], heap[i]
		} else {
			break
		}
	}
}

func fraudScore(heap [5]Neighbor) float32 {
	var fraudCount float32
	for i := 0; i < 5; i++ {
		if heap[i].IsFraud == 1 {
			fraudCount++
		}
	}
	return fraudCount / 5.0
}
