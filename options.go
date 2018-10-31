package lightstep

import (
	"time"

	"github.com/lightstep/lightstep-tracer-go/internal"
	"github.com/lightstep/lightstep-tracer-go/internal/timex"
)

const (
	DefaultReportInterval = time.Second * 3
)

type Option func(*internal.TracerConfig)

func WithReportInterval(reportInterval time.Duration) Option {
	return func(c *internal.TracerConfig) {
		c.ReportInterval = reportInterval
	}
}

func defaultConfig() *internal.TracerConfig {
	return &internal.TracerConfig{
		Client:         internal.NoopClient{},
		ReportInterval: DefaultReportInterval,
		Clock:          timex.NewClock(),
	}
}
