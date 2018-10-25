package lightstep_test

import (
	"context"
	"net"
	"strconv"
	"sync"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	"github.com/lightstep/lightstep-tracer-go/collectorpb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
)

var _ = Describe("gRPCTransport", func() {
	var (
		server    *grpc.Server
		satellite *gRPCSatellite
		options   lightstep.Options
	)

	BeforeEach(func() {
		satellite = newgRPCSatellite()

		server = grpc.NewServer()
		collectorpb.RegisterCollectorServiceServer(server, satellite)

		// Start on a random available port
		listener, err := net.Listen("tcp", "localhost:")
		Expect(err).To(Succeed())

		host, port, err := net.SplitHostPort(listener.Addr().String())
		Expect(err).To(Succeed())

		iPort, err := strconv.Atoi(port)
		Expect(err).To(Succeed())

		options.Collector = lightstep.Endpoint{
			Host:      host,
			Port:      iPort,
			Plaintext: true,
		}

		go server.Serve(listener)
	})

	AfterEach(func() {
		server.Stop()
	})

	testTracer(func() (testSatellite, lightstep.Options) {
		return satellite, options
	})
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
