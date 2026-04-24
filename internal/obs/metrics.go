package obs

import (
	"expvar"
	"sync/atomic"
)

var (
	RangeRequests = expvar.NewInt("range_requests_total")
	RangeFailures = expvar.NewInt("range_failures_total")
	DatagramSent  = expvar.NewInt("datagram_sent_bytes_total")
	StreamSent    = expvar.NewInt("stream_sent_bytes_total")
)

type AtomicGauge struct {
	v atomic.Int64
}

func (g *AtomicGauge) Set(v int64) {
	g.v.Store(v)
}

func (g *AtomicGauge) Load() int64 {
	return g.v.Load()
}
