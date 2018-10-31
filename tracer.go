package lightstep

import (
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
	return nil
}

func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return internal.SpanContext{}, nil
}
