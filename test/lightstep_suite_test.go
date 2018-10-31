package lightstep_test

import (
	"bytes"
	"context"
	"io"

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

var (
	invalidSpanContexts = []opentracing.SpanContext{
		invalidSpanContextMissingTraceID{},
		invalidSpanContextMissingSpanID{},
		invalidSpanContextWrongTraceIDType{},
		invalidSpanContextWrongSpanIDType{},
	}
)

func testTracer(deps *testDependencies) {
	var (
		satellite testSatellite

		subject *lightstep.Tracer
	)

	testReporting := func(reportFn func()) {
		It("successfully serializes and sends finished spans", func() {
			operationName1 := "test-operation-1"
			span := subject.StartSpan(operationName1)
			span.Finish()
			operationName2 := "test-operation-2"
			span = subject.StartSpan(operationName2)
			span.Finish()

			Expect(satellite.ReportedSpans()).To(BeEmpty())

			reportFn()

			Eventually(satellite.ReportedSpans).Should(Not(BeEmpty()))
			Eventually(satellite.ReportedSpans).Should(ContainElement(ReportedSpan{
				OperationName: operationName1,
			}))
			Eventually(satellite.ReportedSpans).Should(ContainElement(ReportedSpan{
				OperationName: operationName2,
			}))
		})

		It("can send multiple span reports", func() {
			operationName := "test1"
			span := subject.StartSpan(operationName)

			span.Finish()
			reportFn()
			Eventually(satellite.ReportedSpans).Should(ContainElement(ReportedSpan{
				OperationName: operationName,
			}))

			operationName = "test2"
			span = subject.StartSpan(operationName)

			span.Finish()
			reportFn()
			Eventually(satellite.ReportedSpans).Should(ContainElement(ReportedSpan{
				OperationName: operationName,
			}))
		})

		It("does not send unfinished spans", func() {
			operationName := "test"
			_ = subject.StartSpan(operationName)

			reportFn()

			Consistently(satellite.ReportedSpans).Should(Not(ContainElement(ReportedSpan{
				OperationName: operationName,
			})))
		})

		It("does not double-send spans that are finished multiple times", func() {
			operationName := "test"
			span := subject.StartSpan(operationName)

			Expect(satellite.ReportedSpans()).To(BeEmpty())

			span.Finish()
			span.Finish()
			reportFn()

			expectedSpan := ReportedSpan{
				OperationName: operationName,
			}
			Eventually(satellite.ReportedSpans).Should(ContainElement(expectedSpan))

			containsDuplicates := func() bool {
				found := false
				for _, s := range satellite.ReportedSpans() {
					if s == expectedSpan {
						if found {
							return true
						}

						found = true
					}
				}

				return false
			}
			Consistently(containsDuplicates).Should(BeFalse())

			span.Finish()
			reportFn()
			Consistently(containsDuplicates).Should(BeFalse())
		})
	}

	BeforeEach(func() {
		satellite = deps.satellite
		subject = lightstep.NewTracer(accessToken, deps.options...)
	})

	It("complies with the OpenTracing standard", func() {
		// Forces a compile-time check
		var tracer opentracing.Tracer
		tracer = subject
		Expect(tracer).NotTo(BeNil())
	})

	Describe("#StartSpan", func() {
		It("returns a non-nil span", func() {
			span := subject.StartSpan("test")
			Expect(span).NotTo(BeNil())
		})
	})

	Describe("#Inject", func() {
		var (
			validSpanContext opentracing.SpanContext
		)

		BeforeEach(func() {
			span := subject.StartSpan("test")
			validSpanContext = span.Context()
		})

		Context("OpenTracing Binary format", func() {
			It("successfully injects a valid SpanContext into an io.Writer", func() {
				err := subject.Inject(validSpanContext, opentracing.Binary, io.Writer(bytes.NewBuffer(nil)))
				Expect(err).To(Succeed())
			})

			It("successfully injects a valid SpanContext into a string", func() {
				carrier := ""
				err := subject.Inject(validSpanContext, opentracing.Binary, &carrier)
				Expect(err).To(Succeed())
			})

			It("successfully injects a valid SpanContext into a byte array", func() {
				err := subject.Inject(validSpanContext, opentracing.Binary, &[]byte{})
				Expect(err).To(Succeed())
			})

			It("fails if the SpanContext is invalid", func() {
				for _, sc := range invalidSpanContexts {
					err := subject.Inject(sc, opentracing.Binary, io.Writer(bytes.NewBuffer(nil)))
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(opentracing.ErrInvalidSpanContext))
				}
			})

			It("fails if the carrier is an unsupported type", func() {
				carrier := make(opentracing.TextMapCarrier)
				err := subject.Inject(validSpanContext, opentracing.Binary, &carrier)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))

				err = subject.Inject(validSpanContext, opentracing.Binary, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))
			})
		})

		Context("OpenTracing TextMap format", func() {
			It("successfully injects a valid SpanContext into an OpenTracing TextMapWriter", func() {
				carrier := make(opentracing.TextMapCarrier)
				err := subject.Inject(validSpanContext, opentracing.TextMap, &carrier)
				Expect(err).To(Succeed())
			})

			It("fails if the SpanContext is invalid", func() {
				for _, sc := range invalidSpanContexts {
					carrier := make(opentracing.TextMapCarrier)
					err := subject.Inject(sc, opentracing.TextMap, &carrier)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(opentracing.ErrInvalidSpanContext))
				}
			})

			It("fails if the carrier is not a TextMapWriter", func() {
				carrier := ""
				err := subject.Inject(validSpanContext, opentracing.TextMap, &carrier)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))

				err = subject.Inject(validSpanContext, opentracing.TextMap, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))
			})
		})

		Context("OpenTracing HTTPHeaders format", func() {
			It("successfully injects a valid SpanContext into an OpenTracing TextMapWriter", func() {
				carrier := make(opentracing.TextMapCarrier)
				err := subject.Inject(validSpanContext, opentracing.HTTPHeaders, &carrier)
				Expect(err).To(Succeed())
			})

			It("fails if the SpanContext is invalid", func() {
				for _, sc := range invalidSpanContexts {
					carrier := make(opentracing.TextMapCarrier)
					err := subject.Inject(sc, opentracing.HTTPHeaders, &carrier)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(opentracing.ErrInvalidSpanContext))
				}
			})

			It("fails if the carrier is not a TextMapWriter", func() {
				carrier := ""
				err := subject.Inject(validSpanContext, opentracing.HTTPHeaders, &carrier)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))

				err = subject.Inject(validSpanContext, opentracing.HTTPHeaders, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))
			})
		})

		It("fails if the format is unsupported", func() {
			err := subject.Inject(validSpanContext, "unknown", io.Writer(bytes.NewBuffer(nil)))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(opentracing.ErrUnsupportedFormat))
		})
	})

	Describe("#Extract", func() {
		var (
			validSpanContext opentracing.SpanContext
		)

		BeforeEach(func() {
			span := subject.StartSpan("test")
			span.SetBaggageItem("baggage", "claim")
			validSpanContext = span.Context()
		})

		Context("OpenTracing Binary format", func() {
			It("successfully extracts from an io.Writer containing a valid SpanContext", func() {
				carrier := io.Writer(bytes.NewBuffer(nil))
				err := subject.Inject(validSpanContext, opentracing.Binary, carrier)
				Expect(err).To(Succeed())

				sc, err := subject.Extract(opentracing.Binary, carrier)
				Expect(err).To(Succeed())
				Expect(sc).To(Equal(validSpanContext))
			})

			It("successfully extracts from a string or *string containing a valid SpanContext", func() {
				carrier := ""
				err := subject.Inject(validSpanContext, opentracing.Binary, &carrier)
				Expect(err).To(Succeed())

				sc, err := subject.Extract(opentracing.Binary, carrier)
				Expect(err).To(Succeed())
				Expect(sc).To(Equal(validSpanContext))

				sc, err = subject.Extract(opentracing.Binary, &carrier)
				Expect(err).To(Succeed())
				Expect(sc).To(Equal(validSpanContext))
			})

			It("successfully extracts from a []byte or *[]byte containing a valid SpanContext", func() {
				carrier := ""
				err := subject.Inject(validSpanContext, opentracing.Binary, &carrier)
				Expect(err).To(Succeed())

				sc, err := subject.Extract(opentracing.Binary, carrier)
				Expect(err).To(Succeed())
				Expect(sc).To(Equal(validSpanContext))

				sc, err = subject.Extract(opentracing.Binary, &carrier)
				Expect(err).To(Succeed())
				Expect(sc).To(Equal(validSpanContext))
			})

			XIt("fails if given an io.Writer containing an invalid SpanContext")

			XIt("fails if given a string or *string containing an invalid SpanContext")

			XIt("fails if given a []byte or *[]byte containing an invalid SpanContext")

			It("fails if the carrier is an unsupported type", func() {
				carrier := make(opentracing.TextMapCarrier)
				_, err := subject.Extract(opentracing.Binary, &carrier)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))
			})
		})

		Context("OpenTracing TextMap format", func() {
			It("successfully extracts from an OpenTracing TextMapReader containing a valid SpanContext", func() {
				carrier := make(opentracing.TextMapCarrier)
				err := subject.Inject(validSpanContext, opentracing.TextMap, &carrier)
				Expect(err).To(Succeed())

				sc, err := subject.Extract(opentracing.TextMap, &carrier)
				Expect(err).To(Succeed())
				Expect(sc).To(Equal(validSpanContext))
			})

			XIt("fails if given a TextMapReader containing an invalid SpanContext")

			It("fails if the carrier is not a TextMapReader", func() {
				carrier := ""
				_, err := subject.Extract(opentracing.TextMap, &carrier)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))

				_, err = subject.Extract(opentracing.TextMap, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))
			})
		})

		Context("OpenTracing HTTPHeaders format", func() {
			It("successfully extracts from an OpenTracing TextMapReader", func() {
				carrier := make(opentracing.TextMapCarrier)
				err := subject.Inject(validSpanContext, opentracing.HTTPHeaders, &carrier)
				Expect(err).To(Succeed())

				sc, err := subject.Extract(opentracing.HTTPHeaders, &carrier)
				Expect(err).To(Succeed())
				Expect(sc).To(Equal(validSpanContext))
			})

			XIt("fails if given a TextMapReader containing an invalid SpanContext")

			It("fails if the carrier is not a TextMapReader", func() {
				carrier := ""
				_, err := subject.Extract(opentracing.HTTPHeaders, &carrier)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))

				_, err = subject.Extract(opentracing.HTTPHeaders, nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(opentracing.ErrInvalidCarrier))
			})
		})

		It("fails if the format is unsupported", func() {
			_, err := subject.Extract("unknown", io.Writer(bytes.NewBuffer(nil)))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(opentracing.ErrUnsupportedFormat))
		})
	})

	Describe("#Flush", func() {
		testReporting(func() {
			err := subject.Flush(context.Background())
			Expect(err).To(Succeed())
		})
	})
}

type testSatellite interface {
	ReportedSpans() []ReportedSpan
}

type ReportedSpan struct {
	OperationName string
}

type testDependencies struct {
	options   []lightstep.Option
	satellite testSatellite
}

type invalidSpanContextMissingTraceID struct {
	SpanID uint64
}

func (sc invalidSpanContextMissingTraceID) ForeachBaggageItem(func(string, string) bool) {}

type invalidSpanContextWrongTraceIDType struct {
	TraceID string
	SpanID  uint64
}

func (sc invalidSpanContextWrongTraceIDType) ForeachBaggageItem(func(string, string) bool) {}

type invalidSpanContextMissingSpanID struct {
	TraceID uint64
}

func (sc invalidSpanContextMissingSpanID) ForeachBaggageItem(func(string, string) bool) {}

type invalidSpanContextWrongSpanIDType struct {
	TraceID uint64
	SpanID  string
}

func (sc invalidSpanContextWrongSpanIDType) ForeachBaggageItem(func(string, string) bool) {}
