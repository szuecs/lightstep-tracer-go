package lightstep_test

import (
	"context"
	"net"
	"sync"

	"github.com/lightstep/lightstep-tracer-common/golang/gogo/collectorpb"
	"github.com/lightstep/lightstep-tracer-go/lightstepgrpc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
)

var _ = Describe("gRPCTransport", func() {
	var (
		deps testDependencies

		server *grpc.Server
	)

	BeforeEach(func() {
		satellite := newgRPCSatellite()
		deps.satellite = satellite

		server = grpc.NewServer()
		collectorpb.RegisterCollectorServiceServer(server, satellite)

		// Start on a random available port
		listener, err := net.Listen("tcp", "localhost:")
		Expect(err).To(Succeed())

		deps.options = append(deps.options, lightstepgrpc.WithGRPC(lightstepgrpc.WithAddress(listener.Addr().String()), lightstepgrpc.WithInsecure()))

		go server.Serve(listener)
	})

	AfterEach(func() {
		server.GracefulStop()
	})

	testTracer(&deps)
})

type gRPCSatellite struct {
	lock          *sync.RWMutex
	reportedSpans []ReportedSpan
}

func newgRPCSatellite() *gRPCSatellite {
	return &gRPCSatellite{
		lock: &sync.RWMutex{},
	}
}

func (s *gRPCSatellite) ReportedSpans() []ReportedSpan {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.reportedSpans
}

func (s *gRPCSatellite) Report(ctx context.Context, req *collectorpb.ReportRequest) (*collectorpb.ReportResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, span := range req.GetSpans() {
		s.reportedSpans = append(s.reportedSpans, ReportedSpan{
			OperationName: span.GetOperationName(),
		})
	}
	return &collectorpb.ReportResponse{}, nil
}
