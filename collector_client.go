package lightstep

import (
	"context"
	"io"
	"net/http"

	"github.com/lightstep/lightstep-tracer-common/golang/protobuf/collectorpb"
)

// Connection describes a closable connection. Exposed for testing.
type Connection interface {
	io.Closer
}

// ConnectorFactory is for testing purposes.
type ConnectorFactory func() (interface{}, Connection, error)

// reportResponse encapsulates internal grpc/http responses.
type reportResponse interface {
	Disable() bool
	DevMode() bool
}

type reportProtoResponse struct {
	*collectorpb.ReportResponse
}

type reportRequest struct {
	protoRequest *collectorpb.ReportRequest
	httpRequest  *http.Request
}

// collectorClient encapsulates internal grpc/http transports.
type collectorClient interface {
	Report(context.Context, reportRequest) (reportResponse, error)
	Translate(context.Context, *reportBuffer) (reportRequest, error)
	ConnectClient() (Connection, error)
	ShouldReconnect() bool
}

func newCollectorClient(opts Options, reporterID uint64, attributes map[string]string) (collectorClient, error) {
	if opts.UseHttp {
		return newHTTPCollectorClient(opts, reporterID, attributes)
	}

	if opts.UseGRPC {
		return newGrpcCollectorClient(opts, reporterID, attributes), nil
	}

	// No transport specified, defaulting to GRPC
	return newGrpcCollectorClient(opts, reporterID, attributes), nil
}

func (c reportProtoResponse) Disable() bool {
	return false
}

func (c reportProtoResponse) DevMode() bool {
	return false
}
