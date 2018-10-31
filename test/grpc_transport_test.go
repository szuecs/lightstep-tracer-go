package lightstep_test

import (
	. "github.com/onsi/ginkgo"
)

var _ = Describe("gRPCTransport", func() {
	var (
		deps testDependencies
	)

	BeforeEach(func() {
		deps.options = append(deps.options)
	})

	testTracer(&deps)
})
