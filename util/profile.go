package util

import (
	"fmt"
	"time"
	"io"
	"runtime"
)

func bToMb(b uint64) float64 {
	return float64(b) / 1024.0 / 1024.0
}

func PrintMemUsage(w io.Writer) {
	var m runtime.MemStats
	now := time.Now()
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "%s,%.4f,%.4f,%.4f,%d\n", now.Format(time.RFC3339Nano), bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)
}
