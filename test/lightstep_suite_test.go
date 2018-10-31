package lightstep_test

import (
	lightstep "github.com/lightstep/lightstep-tracer-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	opentracing "github.com/opentracing/opentracing-go"

	"testing"
)

func TestLightStep(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LightStep Suite")
}

const (
	accessToken = "test access token"
)

func testTracer(deps *testDependencies) {
	var (
		subject *lightstep.Tracer
	)

	BeforeEach(func() {
		subject = lightstep.NewTracer(accessToken, deps.options...)
	})

	It("complies with the OpenTracing standard", func() {
		// Forces a compile-time check
		var tracer opentracing.Tracer
		tracer = subject
		Expect(tracer).NotTo(BeNil())
	})
}

type testDependencies struct {
	options []lightstep.Option
}
