//go:build amd64

package engine

import "unsafe"

//go:noescape
func batchDistAVX2(target *[14]float32, records unsafe.Pointer, numRecords int, distances *float32)
