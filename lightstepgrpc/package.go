package lightstepgrpc

import (
	"context"

	"github.com/lightstep/lightstep-tracer-common/golang/gogo/collectorpb"
	lightstep "github.com/lightstep/lightstep-tracer-go"
	"github.com/lightstep/lightstep-tracer-go/internal"
	"google.golang.org/grpc"
)

const (
	DefaultAddr = "localhost:8080" // TODO
)

func WithGRPC(opts ...Option) (lightstep.Option, error) {
	c := defaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	client, err := dial(c.addr, c.dialOptions...)
	if err != nil {
		return nil, err
	}

	return func(c *internal.TracerConfig) {
		c.Client = client
	}, nil
}

type Option func(*config)

func WithAddress(addr string) Option {
	return func(c *config) {
		c.addr = addr
	}
}

func WithInsecure() Option {
	return func(c *config) {
		c.dialOptions = append(c.dialOptions, grpc.WithInsecure())
	}
}

type config struct {
	addr        string
	dialOptions []grpc.DialOption
}

func defaultConfig() *config {
	return &config{
		addr: DefaultAddr,
	}
}

type client struct {
	conn      *grpc.ClientConn
	satellite collectorpb.CollectorServiceClient
}

func dial(addr string, opts ...grpc.DialOption) (*client, error) {
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &client{
		conn:      conn,
		satellite: collectorpb.NewCollectorServiceClient(conn),
	}, nil
}

func (c *client) Report(ctx context.Context, req internal.ReportRequest) (internal.ReportResponse, error) {
	pReq := &collectorpb.ReportRequest{}
	for _, span := range req.Spans {
		pReq.Spans = append(pReq.Spans, &collectorpb.Span{
			OperationName: span.OperationName,
		})
	}

	var res internal.ReportResponse
	_, err := c.satellite.Report(ctx, pReq)
	return res, err
}
