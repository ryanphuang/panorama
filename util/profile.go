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
	now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "%s,%.4f,%.4f,%.4f,%d\n", now, bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)
}
