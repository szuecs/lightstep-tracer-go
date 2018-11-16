package internal

import (
	"time"

	"github.com/lightstep/lightstep-tracer-go/internal/timex"
)

type TracerConfig struct {
	ClientFactory    func(accessToken string) (Client, error)
	ReportInterval   time.Duration
	MaxBufferedSpans int
	Clock            timex.Clock
}

func WithClock(clock timex.Clock) func(*TracerConfig) {
	return func(c *TracerConfig) {
		c.Clock = clock
	}
}
