package engine

import (
	"math"
	"sync"
	"unsafe"
)

type Neighbor struct {
	DistSq  float32
	IsFraud uint8
}

type ClusterDist struct {
	ID     int
	DistSq float32
}

const NPROBE = 8

var distPool = sync.Pool{
	New: func() any {
		s := make([]float32, 50000)
		return &s
	},
}

// SearchNeighbors uses IVF index to perform an approximate KNN (k=5) search.
func SearchNeighbors(target [14]float32, dataEngine *DataEngine) float32 {
	numClusters := len(dataEngine.Clusters)
	if numClusters == 0 {
		return 0
	}

	// 1. Find top NPROBE clusters using AVX2!
	var dists [2048]float32 // Stack allocated, more than enough for 1024 clusters
	batchDistAVX2_64(&target, unsafe.Pointer(&dataEngine.Clusters[0]), numClusters, &dists[0])

	var topClusters [NPROBE]ClusterDist
	maxClusterDist := float32(math.MaxFloat32)
	for i := 0; i < NPROBE; i++ {
		topClusters[i].DistSq = maxClusterDist
	}

	for i := 0; i < numClusters; i++ {
		d := dists[i]
		if d < topClusters[NPROBE-1].DistSq {
			topClusters[NPROBE-1] = ClusterDist{ID: i, DistSq: d}
			// bubble up
			for j := NPROBE - 1; j > 0; j-- {
				if topClusters[j].DistSq < topClusters[j-1].DistSq {
					topClusters[j], topClusters[j-1] = topClusters[j-1], topClusters[j]
				} else {
					break
				}
			}
		}
	}

	// 2. Search inside the top NPROBE clusters
	heap := initHeap()
	for i := 0; i < NPROBE; i++ {
		clusterID := topClusters[i].ID
		cluster := dataEngine.Clusters[clusterID]
		if cluster.Count == 0 {
			continue
		}
		
		start := int(cluster.Start)
		end := start + int(cluster.Count)
		searchChunkInline(target, dataEngine.Records[start:end], &heap)
	}

	return fraudScore(heap)
}

func searchChunkInline(target [14]float32, records []VectorRecord, heap *[5]Neighbor) {
	if len(records) == 0 {
		return
	}

	sptr := distPool.Get().(*[]float32)
	dists := *sptr
	if len(dists) < len(records) {
		dists = make([]float32, len(records)*2)
		*sptr = dists
	}

	// Calculate all distances in batch using AVX2 Assembly!
	batchDistAVX2(&target, unsafe.Pointer(&records[0]), len(records), &dists[0])

	for i := 0; i < len(records); i++ {
		d := dists[i]
		if d < heap[4].DistSq {
			heap[4] = Neighbor{DistSq: d, IsFraud: records[i].IsFraud}
			bubbleUp(heap)
		}
	}

	distPool.Put(sptr)
}

func initHeap() [5]Neighbor {
	var heap [5]Neighbor
	maxDist := float32(math.MaxFloat32)
	for i := 0; i < 5; i++ {
		heap[i].DistSq = maxDist
	}
	return heap
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
