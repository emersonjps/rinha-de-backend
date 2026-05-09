package engine

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const RecordSize = 60 // 14*4 + 1 + 3 bytes padding

type VectorRecord struct {
	Dimensions [14]float32
	IsFraud    uint8
	_          [3]byte
}

type DataEngine struct {
	File    *os.File
	Data    []byte
	Records []VectorRecord
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
	if size == 0 {
		_ = file.Close()
		return nil, fmt.Errorf("references file is empty")
	}
	if size%RecordSize != 0 {
		_ = file.Close()
		return nil, fmt.Errorf("invalid references size: %d is not a multiple of %d", size, RecordSize)
	}

	maxInt := int64(^uint(0) >> 1)
	if size > maxInt {
		_ = file.Close()
		return nil, fmt.Errorf("references file too large to map: %d", size)
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("mmap references file: %w", err)
	}

	recordCount := int(size / RecordSize)
	records := unsafe.Slice((*VectorRecord)(unsafe.Pointer(&data[0])), recordCount)

	return &DataEngine{
		File:    file,
		Data:    data,
		Records: records,
	}, nil
}

// NewDataEngine is a convenience alias for LoadEngine.
func NewDataEngine(path string) (*DataEngine, error) {
	return LoadEngine(path)
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
