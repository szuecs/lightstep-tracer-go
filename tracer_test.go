package lightstep_test

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	. "github.com/lightstep/lightstep-tracer-go"
	cpb "github.com/lightstep/lightstep-tracer-go/collectorpb"
	cpbfakes "github.com/lightstep/lightstep-tracer-go/collectorpb/collectorpbfakes"
	"github.com/lightstep/lightstep-tracer-go/lightstepfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tracer", func() {
	var tracer Tracer
	var opts Options
	var fakeClient *cpbfakes.FakeCollectorServiceClient
	var fakeConn ConnectorFactory
	var fakeRecorder *lightstepfakes.FakeSpanRecorder
	const fakeAccessToken = "YOU SHALL NOT PASS"
	var eventHandler func(Event)
	var eventChan <-chan Event

	BeforeEach(func() {
		opts = Options{}
		fakeClient = new(cpbfakes.FakeCollectorServiceClient)
		fakeClient.ReportReturns(&cpb.ReportResponse{}, nil)
		fakeConn = fakeGrpcConnection(fakeClient)
		fakeRecorder = new(lightstepfakes.FakeSpanRecorder)
		eventHandler, eventChan = NewOnEventChannel(10)
	})

	JustBeforeEach(func() {
		tracer = NewTracer(opts)
	})

	AfterEach(func() {
		closeTestTracer(tracer)
	})

	Describe("Flush", func() {
		BeforeEach(func() {
			opts = Options{
				AccessToken: fakeAccessToken,
				ConnFactory: fakeConn,
				OnEvent:     eventHandler,
			}
		})

		Context("when the tracer is dropping spans", func() {
			const ExpectedDroppedSpans = 2

			JustBeforeEach(func() {
				// overflow the span buffer so the tracer starts to drop spans
				for i := 0; i < DefaultMaxSpans+ExpectedDroppedSpans; i++ {
					tracer.StartSpan(fmt.Sprint("span ", i)).Finish()
				}
			})

			It("OnEvent emits EventStatusReport", func(done Done) {
				tracer.Flush(context.Background())

				err := <-eventChan
				dropSpanErr, ok := err.(EventStatusReport)
				Expect(ok).To(BeTrue())

				Expect(dropSpanErr.DroppedSpans()).To(Equal(ExpectedDroppedSpans))
				close(done)
			})

			It("OnEvent emits exactly one event", func() {
				tracer.Flush(context.Background())

				Eventually(eventChan).Should(Receive())
				Consistently(eventChan).ShouldNot(Receive())
			})
		})

		Context("when the tracer is disabled", func() {
			JustBeforeEach(func() {
				tracer.Disable()
			})

			It("should not flush spans", func() {
				reportCallCount := fakeClient.ReportCallCount()
				tracer.StartSpan("these spans should not be recorded").Finish()
				tracer.StartSpan("or flushed").Finish()

				tracer.Flush(context.Background())

				Consistently(fakeClient.ReportCallCount).Should(Equal(reportCallCount))
			}, 5)

			It("OnEvent emits EventFlushError", func(done Done) {
				tracer.Flush(context.Background())

				event := <-eventChan
				flushErrorEvent, ok := event.(EventFlushError)
				Expect(ok).To(BeTrue())
				Expect(flushErrorEvent.State()).To(Equal(FlushErrorTracerDisabled))
				close(done)
			})

			It("OnEvent emits exactly one event", func() {
				tracer.Flush(context.Background())

				Eventually(eventChan).Should(Receive())
				Consistently(eventChan).ShouldNot(Receive())
			})
		})

		Context("when tracer has been closed", func() {
			var reportCallCount int

			JustBeforeEach(func() {
				tracer.Close(context.Background())
				reportCallCount = fakeClient.ReportCallCount()
			})

			It("should not flush spans", func() {
				tracer.StartSpan("can't flush this").Finish()
				tracer.StartSpan("hammer time").Finish()

				tracer.Flush(context.Background())

				Consistently(fakeClient.ReportCallCount).Should(Equal(reportCallCount))
			})

			It("OnEvent emits EventFlushError", func(done Done) {
				tracer.Flush(context.Background())

				Eventually(eventChan).Should(Receive())

				event := <-eventChan
				flushErrorEvent, ok := event.(EventFlushError)
				Expect(ok).To(BeTrue())
				Expect(flushErrorEvent.State()).To(Equal(FlushErrorTracerClosed))
				close(done)
			})

			It("OnEvent emits exactly two events", func() {
				tracer.Flush(context.Background())

				Eventually(eventChan).Should(Receive())
				Eventually(eventChan).Should(Receive())
				Consistently(eventChan).ShouldNot(Receive())
			})
		})

		Context("when a report is in progress", func() {
			var startReportonce sync.Once
			var startReportch chan struct{}
			var finishReportch chan struct{}

			BeforeEach(func() {
				finishReportch = make(chan struct{})
				startReportch = make(chan struct{})
				fakeClient.ReportStub = func(ctx context.Context, req *cpb.ReportRequest, options ...grpc.CallOption) (*cpb.ReportResponse, error) {
					startReportonce.Do(func() { close(startReportch) })
					<-finishReportch
					return new(cpb.ReportResponse), nil
				}
			})

			JustBeforeEach(func() {
				// wait for tracer to have a report in flight
				Eventually(startReportch).Should(BeClosed())
			})

			It("retries flushing", func() {
				tracer.StartSpan("these spans should sit in the buffer").Finish()
				tracer.StartSpan("while the last report is in flight").Finish()

				flushFinishedch := make(chan struct{})
				go func() {
					Flush(context.Background(), tracer)
					close(flushFinishedch)
				}()
				// flush should wait for the last report to finish
				Consistently(flushFinishedch).ShouldNot(BeClosed())
				// no spans should have been reported yet
				Expect(getReportedGRPCSpans(fakeClient)).To(HaveLen(0))
				// allow the last report to finish
				close(finishReportch)
				// flush should now send a report with the last two spans
				Eventually(flushFinishedch).Should(BeClosed())
				// now the spans that were sitting in the buffer should be reported
				Expect(getReportedGRPCSpans(fakeClient)).Should(HaveLen(2))
			})
		})
	})

	Context("Close", func() {

		BeforeEach(func() {
			opts = Options{
				AccessToken:        fakeAccessToken,
				ConnFactory:        fakeConn,
				MinReportingPeriod: 100 * time.Second,
			}
		})

		It("flushes the buffer before closing", func() {
			tracer.StartSpan("span").Finish()
			err := CloseTracer(tracer)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeClient.ReportCallCount()).To(Equal(1))
			tracer.StartSpan("span2").Finish()
			Consistently(fakeClient.ReportCallCount).Should(Equal(1))
		})
	})

	Context("When tracer has a SpanRecorder", func() {
		BeforeEach(func() {
			opts = Options{
				AccessToken: fakeAccessToken,
				ConnFactory: fakeConn,
				Recorder:    fakeRecorder,
			}
		})

		It("calls RecordSpan after finishing a span", func() {
			tracer.StartSpan("span").Finish()
			Expect(fakeRecorder.RecordSpanCallCount()).ToNot(BeZero())
		})
	})

	Context("When tracer does not have a SpanRecorder", func() {
		BeforeEach(func() {
			opts = Options{
				AccessToken: fakeAccessToken,
				ConnFactory: fakeConn,
				Recorder:    nil,
			}
		})

		It("doesn't call RecordSpan after finishing a span", func() {
			span := tracer.StartSpan("span")
			Expect(span.Finish).ToNot(Panic())
		})
	})
})
