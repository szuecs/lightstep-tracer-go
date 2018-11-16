package lightstepgrpc

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/lightstep/lightstep-tracer-common/golang/gogo/collectorpb"
	lightstep "github.com/lightstep/lightstep-tracer-go"
	"github.com/lightstep/lightstep-tracer-go/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	DefaultAddr = "localhost:8080" // TODO
)

func WithGRPC(opts ...Option) lightstep.Option {
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

	addr := c.addr

	return func(tc *internal.TracerConfig) {
		tc.ClientFactory = func(accessToken string) (internal.Client, error) {
			conn, err := grpc.Dial(addr, dialOptions...)
			if err != nil {
				return nil, err
			}

			return &client{
				accessToken: accessToken,
				conn:        conn,
				satellite:   collectorpb.NewCollectorServiceClient(conn),
			}, nil
		}
	}
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
	accessToken string
	conn        *grpc.ClientConn
	satellite   collectorpb.CollectorServiceClient
}

func (c *client) Report(ctx context.Context, req internal.ReportRequest) (internal.ReportResponse, error) {
	pReq := &collectorpb.ReportRequest{
		Auth: &collectorpb.Auth{
			AccessToken: c.accessToken,
		},
		Reporter: &collectorpb.Reporter{
			ReporterId: 1,
		},
	}

	pReq.Reporter.Tags = append(pReq.Reporter.Tags, &collectorpb.KeyValue{
		Key:   "lightstep.component_name",
		Value: &collectorpb.KeyValue_StringValue{StringValue: "test"},
	})
	for _, span := range req.Spans {
		end := time.Now()
		start := end.Add(time.Second * -1)
		pReq.Spans = append(pReq.Spans, &collectorpb.Span{
			OperationName: span.OperationName,
			SpanContext: &collectorpb.SpanContext{
				TraceId: 123,
				SpanId:  456,
			},
			StartTimestamp: &types.Timestamp{
				Seconds: start.Unix(),
				Nanos:   int32(start.Nanosecond()),
			},
			DurationMicros: uint64(time.Second / time.Microsecond),
		})
	}

	var res internal.ReportResponse
	_, err := c.satellite.Report(ctx, pReq)
	return res, err
}

func (c *client) Close(context.Context) error {
	err := c.conn.Close()
	if err != grpc.ErrClientConnClosing {
		return err
	}
	return nil
}
