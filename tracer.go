package lightstep

import (
	"context"
	"io"
	"sync"

	"github.com/lightstep/lightstep-tracer-go/internal"
	"github.com/lightstep/lightstep-tracer-go/internal/timex"
	opentracing "github.com/opentracing/opentracing-go"
)

type Tracer struct {
	lock             *sync.Mutex
	closed           bool
	ticker           timex.Ticker
	maxBufferedSpans int
	spans            []internal.Span
	recorder         internal.RecordSpan
	client           internal.Client
}

func NewTracer(accessToken string, opts ...Option) (*Tracer, error) {
	c := defaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	client, err := c.ClientFactory(accessToken)
	if err != nil {
		return nil, err
	}

	t := &Tracer{
		lock:             &sync.Mutex{},
		maxBufferedSpans: c.MaxBufferedSpans,
		client:           client,
	}
	t.recorder = func(span internal.Span) {
		t.lock.Lock()
		defer t.lock.Unlock()

		if !t.closed {
			t.spans = append(t.spans, span)

			if len(t.spans) == t.maxBufferedSpans {
				go t.runReport(context.Background())
			}
		}
	}

	if c.ReportInterval != 0 {
		t.ticker = c.Clock.NewTicker(c.ReportInterval)
	}
	t.runLoop()

	return t, nil
}

func (t *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return internal.NewSpan(operationName, internal.WithRecorder(t.recorder))
}

func (t *Tracer) Inject(spanContext opentracing.SpanContext, format interface{}, carrier interface{}) error {
	_, ok := spanContext.(internal.SpanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}

	switch format {
	case opentracing.Binary:
		switch carrier.(type) {
		case io.Writer, *string, *[]byte:
		default:
			return opentracing.ErrInvalidCarrier
		}
	case opentracing.TextMap, opentracing.HTTPHeaders:
		switch carrier.(type) {
		case opentracing.TextMapWriter:
		default:
			return opentracing.ErrInvalidCarrier
		}
	default:
		return opentracing.ErrUnsupportedFormat
	}

	return nil
}

func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.Binary:
		switch carrier.(type) {
		case io.Writer, string, *string, []byte, *[]byte:
		default:
			return nil, opentracing.ErrInvalidCarrier
		}
	case opentracing.TextMap, opentracing.HTTPHeaders:
		switch carrier.(type) {
		case opentracing.TextMapReader:
		default:
			return nil, opentracing.ErrInvalidCarrier
		}
	default:
		return nil, opentracing.ErrUnsupportedFormat
	}

	return internal.SpanContext{}, nil
}

func (t *Tracer) Flush(ctx context.Context) error {
	t.runReport(ctx)
	return nil
}

func (t *Tracer) Close(ctx context.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.closed = true

	if t.ticker != nil {
		t.ticker.Stop()
	}

	return t.client.Close(ctx)
}

func (t *Tracer) runReport(ctx context.Context) {
	t.lock.Lock()

	var req internal.ReportRequest

	for _, span := range t.spans {
		req.Spans = append(req.Spans, internal.SpanRequest{
			OperationName: span.OperationName(),
		})
	}
	t.spans = nil

	t.lock.Unlock()

	t.client.Report(ctx, req) // TODO: handle error
}

func (t *Tracer) runLoop() {
	if t.ticker != nil {
		go func() {
			for range t.ticker.C() {
				go t.runReport(context.Background())
			}
		}()
	}
}
