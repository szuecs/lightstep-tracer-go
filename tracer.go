package lightstep

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	"runtime"
	"sync"

	ot "github.com/opentracing/opentracing-go"
)

// Implements the `Tracer` interface. Buffers spans and forwards the to a Lightstep collector.
type tracerImpl struct {
	//////////////////////////////////////////////////////////////
	// IMMUTABLE IMMUTABLE IMMUTABLE IMMUTABLE IMMUTABLE IMMUTABLE
	//////////////////////////////////////////////////////////////

	// Note: there may be a desire to update some of these fields
	// at runtime, in which case suitable changes may be needed
	// for variables accessed during Flush.

	reporterID       uint64 // the LightStep tracer guid
	opts             Options
	textPropagator   textMapPropagator
	binaryPropagator lightstepBinaryPropagator

	//////////////////////////////////////////////////////////
	// MUTABLE MUTABLE MUTABLE MUTABLE MUTABLE MUTABLE MUTABLE
	//////////////////////////////////////////////////////////

	// the following fields are modified under `lock`.
	lock sync.Mutex

	// Remote service that will receive reports.
	client       collectorClient
	conn         Connection
	closech      chan struct{}
	reportLoopch chan struct{}

	// Two buffers of data.
	buffer   reportBuffer
	flushing reportBuffer

	// Flush state.
	flushingLock      sync.Mutex
	reportInFlight    bool
	lastReportAttempt time.Time

	// We allow our remote peer to disable this instrumentation at any
	// time, turning all potentially costly runtime operations into
	// no-ops.
	//
	// TODO this should use atomic load/store to test disabled
	// prior to taking the lock, do please.
	disabled bool
}

// NewTracer creates and starts a new Lightstep Tracer.
func NewTracer(opts Options) Tracer {
	err := opts.Initialize()
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	attributes := map[string]string{}
	for k, v := range opts.Tags {
		attributes[k] = fmt.Sprint(v)
	}
	// Don't let the GrpcOptions override these values. That would be confusing.
	attributes[TracerPlatformKey] = TracerPlatformValue
	attributes[TracerPlatformVersionKey] = runtime.Version()
	attributes[TracerVersionKey] = TracerVersionValue

	now := time.Now()
	impl := &tracerImpl{
		opts:       opts,
		reporterID: genSeededGUID(),
		buffer:     newSpansBuffer(opts.MaxBufferedSpans),
		flushing:   newSpansBuffer(opts.MaxBufferedSpans),
	}

	impl.buffer.setCurrent(now)

	if opts.UseThrift {
		impl.client = newThriftCollectorClient(opts, impl.reporterID, attributes)
	} else {
		impl.client = newGrpcCollectorClient(opts, impl.reporterID, attributes)
	}

	conn, err := impl.client.ConnectClient()
	if err != nil {
		impl.onError(err)
		return nil
	}

	impl.conn = conn
	impl.closech = make(chan struct{})
	impl.reportLoopch = make(chan struct{})

	// Important! incase close is called before go routine is kicked off
	closech := impl.closech
	go func() {
		impl.reportLoop(closech)
		close(impl.reportLoopch)
	}()

	return impl
}

func (t *tracerImpl) Options() Options {
	return t.opts
}

func (t *tracerImpl) StartSpan(
	operationName string,
	sso ...ot.StartSpanOption,
) ot.Span {
	return newSpan(operationName, t, sso)
}

func (t *tracerImpl) Inject(sc ot.SpanContext, format interface{}, carrier interface{}) error {
	switch format {
	case ot.TextMap, ot.HTTPHeaders:
		return t.textPropagator.Inject(sc, carrier)
	case BinaryCarrier:
		return t.binaryPropagator.Inject(sc, carrier)
	}
	return ot.ErrUnsupportedFormat
}

func (t *tracerImpl) Extract(format interface{}, carrier interface{}) (ot.SpanContext, error) {
	switch format {
	case ot.TextMap, ot.HTTPHeaders:
		return t.textPropagator.Extract(carrier)
	case BinaryCarrier:
		return t.binaryPropagator.Extract(carrier)
	}
	return nil, ot.ErrUnsupportedFormat
}

func (t *tracerImpl) reconnectClient(now time.Time) {
	conn, err := t.client.ConnectClient()
	if err != nil {
		t.onError(err)
	} else {
		t.lock.Lock()
		oldConn := t.conn
		t.conn = conn
		t.lock.Unlock()

		oldConn.Close()
		maybeLogInfof("reconnected client connection", t.opts.Verbose)
	}
}

// Close flushes and then terminates the LightStep collector.
func (t *tracerImpl) Close(ctx context.Context) {
	t.lock.Lock()
	closech := t.closech
	t.closech = nil
	t.lock.Unlock()

	if closech != nil {
		// notify report loop that we are closing
		close(closech)

		// wait for report loop to finish
		if t.reportLoopch != nil {
			select {
			case <-t.reportLoopch:
				t.Flush(ctx)
			case <-ctx.Done():
				return
			}
		}
	}

	// now its safe to close the connection
	t.lock.Lock()
	conn := t.conn
	t.conn = nil
	t.reportLoopch = nil
	t.lock.Unlock()

	if conn != nil {
		err := conn.Close()
		if err != nil {
			t.onError(err)
		}
	}
}

// RecordSpan records a finished Span.
func (t *tracerImpl) RecordSpan(raw RawSpan) {
	t.lock.Lock()

	// Early-out for disabled runtimes
	if t.disabled {
		t.lock.Unlock()
		return
	}

	t.buffer.addSpan(raw)
	t.lock.Unlock()

	if t.opts.Recorder != nil {
		t.opts.Recorder.RecordSpan(raw)
	}
}

// Flush sends all buffered data to the collector.
func (t *tracerImpl) Flush(ctx context.Context) {
	t.flushingLock.Lock()
	defer t.flushingLock.Unlock()

	err := t.preFlush()
	if err != nil {
		t.onError(err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, t.opts.ReportTimeout)
	defer cancel()
	resp, err := t.client.Report(ctx, &t.flushing)

	respErrs := resp.GetErrors()
	if err == nil && len(respErrs) > 0 {
		err = fmt.Errorf(respErrs[0])
	}

	t.postFlush(err)

	if resp.Disable() {
		t.Disable()
	}

	if err != nil {
		t.onError(err)
	}
}

func (t *tracerImpl) preFlush() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.disabled {
		return newDisabledError(ErrFlushingWhenDisabled)
	}

	if t.conn == nil {
		return newClosedError(ErrFlushingWhenClosed)
	}

	now := time.Now()
	t.buffer, t.flushing = t.flushing, t.buffer
	t.reportInFlight = true
	t.flushing.setFlushing(now)
	t.buffer.setCurrent(now)
	t.lastReportAttempt = now
	return nil
}

func (t *tracerImpl) postFlush(err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.reportInFlight = false
	if err != nil {
		// Restore the records that did not get sent correctly
		t.buffer.mergeFrom(&t.flushing)
		return
	}

	//
	droppedSent := t.flushing.droppedSpanCount
	if droppedSent != 0 {
		maybeLogInfof("client reported %d dropped spans", t.opts.Verbose, droppedSent)
	}
	t.flushing.clear()
}

func (t *tracerImpl) Disable() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.disabled {
		return
	}

	fmt.Printf("Disabling Runtime instance: %p", t)

	t.buffer.clear()
	t.disabled = true
}

func (t *tracerImpl) onError(err error) {
	if t.opts.OnError != nil {
		t.opts.OnError(err)
	}
}

// Every MinReportingPeriod the reporting loop wakes up and checks to see if
// either (a) the Runtime's max reporting period is about to expire (see
// maxReportingPeriod()), (b) the number of buffered log records is
// approaching kMaxBufferedLogs, or if (c) the number of buffered span records
// is approaching kMaxBufferedSpans. If any of those conditions are true,
// pending data is flushed to the remote peer. If not, the reporting loop waits
// until the next cycle. See Runtime.maybeFlush() for details.
//
// This could alternatively be implemented using flush channels and so forth,
// but that would introduce opportunities for client code to block on the
// runtime library, and we want to avoid that at all costs (even dropping data,
// which can certainly happen with high data rates and/or unresponsive remote
// peers).
func (t *tracerImpl) shouldFlushLocked(now time.Time) bool {
	if now.Add(t.opts.MinReportingPeriod).Sub(t.lastReportAttempt) > t.opts.ReportingPeriod {
		// Flush timeout.
		maybeLogInfof("--> timeout", t.opts.Verbose)
		return true
	} else if t.buffer.isHalfFull() {
		// Too many queued span records.
		maybeLogInfof("--> span queue", t.opts.Verbose)
		return true
	}
	return false
}

func (t *tracerImpl) reportLoop(closech chan struct{}) {
	tickerChan := time.Tick(t.opts.MinReportingPeriod)
	for {
		select {
		case <-tickerChan:
			now := time.Now()

			t.lock.Lock()
			disabled := t.disabled
			reconnect := !t.reportInFlight && t.client.ShouldReconnect()
			shouldFlush := t.shouldFlushLocked(now)
			t.lock.Unlock()

			if disabled {
				return
			}
			if shouldFlush {
				t.Flush(context.Background())
			}
			if reconnect {
				t.reconnectClient(now)
			}
		case <-closech:
			return
		}
	}
}
