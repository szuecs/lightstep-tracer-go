package lightstepgrpc

import (
	"context"
	"crypto/tls"

	"github.com/lightstep/lightstep-tracer-common/golang/gogo/collectorpb"
	lightstep "github.com/lightstep/lightstep-tracer-go"
	"github.com/lightstep/lightstep-tracer-go/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	DefaultAddr = "localhost:8080" // TODO
)

func WithGRPC(opts ...Option) (lightstep.Option, error) {
	c := defaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	var dialOptions []grpc.DialOption
	if c.insecure {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(c.tlsConfig)))
	}

	client, err := dial(c.addr, dialOptions...)
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
		c.insecure = true
	}
}

type config struct {
	addr      string
	insecure  bool
	tlsConfig *tls.Config
}

func defaultConfig() *config {
	return &config{
		addr:      DefaultAddr,
		tlsConfig: &tls.Config{},
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
