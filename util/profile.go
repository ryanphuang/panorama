package util

import (
	"fmt"
	"io"
	"runtime"
)

func bToMb(b uint64) float64 {
	return float64(b) / 1024.0 / 1024.0
}

func PrintMemUsage(w io.Writer) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "%.4f,%.4f,%.4f,%d\n", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)
}
