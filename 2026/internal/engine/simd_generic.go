//go:build !amd64

package engine

import "unsafe"

// Fallback for non-amd64 architectures (e.g. arm64 on Mac M1/M2)
func batchDistAVX2(target *[14]float32, records unsafe.Pointer, numRecords int, distances *float32) {
	// Fallback uses the inline function that we already have
	recs := unsafe.Slice((*VectorRecord)(records), numRecords)
	dists := unsafe.Slice(distances, numRecords)
	
	for i := 0; i < numRecords; i++ {
		rec := &recs[i]

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
			
		dists[i] = distSq
	}
}
