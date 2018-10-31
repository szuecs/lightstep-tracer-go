package lightstep

import (
	"io"

	"github.com/lightstep/lightstep-tracer-go/internal"
	opentracing "github.com/opentracing/opentracing-go"
)

type Tracer struct{}

func NewTracer(accessToken string, opts ...Option) *Tracer {
	return &Tracer{}
}

func (t *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return internal.NewSpan(operationName)
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
