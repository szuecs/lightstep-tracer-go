package lightstep

import (
	"time"

	"github.com/lightstep/lightstep-tracer-go/internal"
	"github.com/lightstep/lightstep-tracer-go/internal/timex"
)

const (
	DefaultReportInterval   = time.Second * 3
	DefaultMaxBufferedSpans = 1000
)

type Option func(*internal.TracerConfig)

func WithReportInterval(reportInterval time.Duration) Option {
	return func(c *internal.TracerConfig) {
		c.ReportInterval = reportInterval
	}
}

func WithMaxBuffedSpans(maxBufferedSpans int) Option {
	return func(c *internal.TracerConfig) {
		c.MaxBufferedSpans = maxBufferedSpans
	}
}

func defaultConfig() *internal.TracerConfig {
	return &internal.TracerConfig{
		ClientFactory: func(string) (internal.Client, error) {
			return internal.NoopClient{}, nil
		},
		ReportInterval:   DefaultReportInterval,
		MaxBufferedSpans: DefaultMaxBufferedSpans,
		Clock:            timex.NewClock(),
	}
}
