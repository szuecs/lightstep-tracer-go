package metrics_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/lightstep/lightstep-tracer-common/golang/gogo/metricspb"
	"github.com/lightstep/lightstep-tracer-go/internal/metrics"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reporter", func() {
	var reporter *metrics.Reporter
	var ingestRequest metricspb.IngestRequest

	JustBeforeEach(func() {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := ioutil.ReadAll(r.Body)
			err := proto.Unmarshal(body, &ingestRequest)
			if !Expect(err).To(BeNil()) {
				return
			}
		})
		s := httptest.NewServer(h)
		url := fmt.Sprintf("http://%s", s.Listener.Addr().String())
		reporter = metrics.NewReporter(
			metrics.WithReporterAddress(url),
		)
	})
	Describe("Measure", func() {
		It("should return an IngestRequest", func() {
			// metric := metrics.Metrics{}
			err := reporter.Measure(context.Background())
			if !Expect(err).To(BeNil()) {
				return
			}
			// check metrics are present
			Expect(ingestRequest.GetPoints()).To(HaveLen(11))

			// TODO: check that we're sending a delta
		})
	})
})

func TestLightstepMetricsGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LightstepMetricsGo Suite")
}
