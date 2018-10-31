package lightstep

import "github.com/lightstep/lightstep-tracer-go/internal"

type Option func(*internal.TracerConfig)

func defaultConfig() *internal.TracerConfig {
	return &internal.TracerConfig{
		Client: internal.NoopClient{},
	}
}
