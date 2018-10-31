package lightstepgrpc

import (
	lightstep "github.com/lightstep/lightstep-tracer-go"
	"github.com/lightstep/lightstep-tracer-go/internal"
	"google.golang.org/grpc"
)

func WithGRPC(opts ...Option) (lightstep.Option, error) {
	return func(c *internal.TracerConfig) {}, nil
}

type Option func(*config)

type config struct {
	addr        string
	dialOptions []grpc.DialOption
}
