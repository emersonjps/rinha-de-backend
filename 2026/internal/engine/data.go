package engine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"syscall"
	"unsafe"
)

const RecordSize = 60 // 14*4 + 1 + 3 padding
const HeaderSize = 8  // Magic (4) + NumClusters (4)
const ClusterInfoSize = 64 // 14*4 + Start (4) + Count (4)

type VectorRecord struct {
	Dimensions [14]float32
	IsFraud    uint8
	_          [3]byte
}

type Cluster struct {
	Centroid [14]float32
	Start    uint32
	Count    uint32
}

type DataEngine struct {
	File     *os.File
	Data     []byte
	Clusters []Cluster
	Records  []VectorRecord
}

func LoadEngine(path string) (*DataEngine, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open references file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("stat references file: %w", err)
	}

	size := info.Size()
	if size < HeaderSize {
		_ = file.Close()
		return nil, fmt.Errorf("file too small")
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("mmap references file: %w", err)
	}

	if !bytes.Equal(data[:4], []byte("IVF1")) {
		_ = syscall.Munmap(data)
		_ = file.Close()
		return nil, fmt.Errorf("invalid magic bytes, expected IVF1")
	}

	numClusters := binary.LittleEndian.Uint32(data[4:8])
	clustersOffset := HeaderSize
	recordsOffset := HeaderSize + int(numClusters)*ClusterInfoSize

	if size < int64(recordsOffset) {
		_ = syscall.Munmap(data)
		_ = file.Close()
		return nil, fmt.Errorf("file size does not match cluster count")
	}

	clusters := make([]Cluster, numClusters)
	for i := 0; i < int(numClusters); i++ {
		offset := clustersOffset + i*ClusterInfoSize
		for j := 0; j < 14; j++ {
			bits := binary.LittleEndian.Uint32(data[offset+j*4 : offset+j*4+4])
			clusters[i].Centroid[j] = math.Float32frombits(bits)
		}
		clusters[i].Start = binary.LittleEndian.Uint32(data[offset+56 : offset+60])
		clusters[i].Count = binary.LittleEndian.Uint32(data[offset+60 : offset+64])
	}

	recordDataSize := size - int64(recordsOffset)
	if recordDataSize%RecordSize != 0 {
		_ = syscall.Munmap(data)
		_ = file.Close()
		return nil, fmt.Errorf("invalid records area size")
	}

	recordCount := int(recordDataSize / RecordSize)
	records := unsafe.Slice((*VectorRecord)(unsafe.Pointer(&data[recordsOffset])), recordCount)

	return &DataEngine{
		File:     file,
		Data:     data,
		Clusters: clusters,
		Records:  records,
	}, nil
}

func (e *DataEngine) Close() {
	if e == nil {
		return
	}
	if e.Data != nil {
		_ = syscall.Munmap(e.Data)
		e.Data = nil
		e.Records = nil
	}
	if e.File != nil {
		_ = e.File.Close()
		e.File = nil
	}
}
