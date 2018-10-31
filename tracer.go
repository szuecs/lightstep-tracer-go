package lightstep

import (
	"context"
	"io"
	"sync"

	"github.com/lightstep/lightstep-tracer-go/internal"
	opentracing "github.com/opentracing/opentracing-go"
)

type Tracer struct {
	lock     *sync.Mutex
	closed   bool
	spans    []internal.Span
	recorder internal.RecordSpan
	client   internal.Client
}

func NewTracer(accessToken string, opts ...Option) *Tracer {
	c := defaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	t := &Tracer{
		lock:   &sync.Mutex{},
		client: c.Client,
	}
	t.recorder = func(span internal.Span) {
		t.lock.Lock()
		defer t.lock.Unlock()

		if !t.closed {
			t.spans = append(t.spans, span)
		}
	}

	return t
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
	return t.runReport(ctx)
}

func (t *Tracer) Close(ctx context.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.closed = true

	return nil
}

func (t *Tracer) runReport(ctx context.Context) error {
	t.lock.Lock()

	var req internal.ReportRequest

	for _, span := range t.spans {
		req.Spans = append(req.Spans, internal.SpanRequest{
			OperationName: span.OperationName(),
		})
	}
	t.spans = nil

	t.lock.Unlock()

	_, err := t.client.Report(ctx, req)
	return err
}
