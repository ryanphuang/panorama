package service

import (
	"fmt"
	"testing"
	"time"

	"deephealth/service"
	"deephealth/store"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	portstart = 10000
	portend   = 30000
)

func BenchmarkAddReport(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {

	}
}

func TestMain(m *testing.M) {
	port := portstart + int(r.Intn(portend-portstart))
	addr := fmt.Sprintf("localhost:%d", port)
	storage := store.NewRawHealthStorage("TS_1", "TS_2", "TS_3", "TS_4")
	hs := service.NewHealthService(addr, "XFE_1", storage)
	hs.Start()

}
