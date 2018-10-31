package internal

import (
	"time"

	"github.com/lightstep/lightstep-tracer-go/internal/timex"
)

type TracerConfig struct {
	Client         Client
	ReportInterval time.Duration
	Clock          timex.Clock
}

func WithClock(clock timex.Clock) func(*TracerConfig) {
	return func(c *TracerConfig) {
		c.Clock = clock
	}
}
