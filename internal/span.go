package internal

import (
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type Span struct {
	operationName string
	ctx           SpanContext
	tracer        opentracing.Tracer
	recorder      RecordSpan
	finishOnce    *sync.Once
}

func NewSpan(operationName string, opts ...SpanOption) Span {
	c := defaultSpanConfig()
	for _, opt := range opts {
		opt(c)
	}

	return Span{
		operationName: operationName,
		recorder:      c.recorder,
		finishOnce:    &sync.Once{},
	}
}

func (s Span) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

func (s Span) FinishWithOptions(opts opentracing.FinishOptions) {
	s.finishOnce.Do(func() {
		s.recorder(s)
	})
}

func (s Span) Context() opentracing.SpanContext {
	return s.ctx
}

func (s Span) SetOperationName(operationName string) opentracing.Span {
	s.operationName = operationName
	return s
}

func (s Span) SetTag(key string, value interface{}) opentracing.Span {
	return s
}

func (s Span) LogFields(fields ...log.Field) {}

func (s Span) LogKV(alternatingKeyValues ...interface{}) {}

func (s Span) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	return s
}

func (s Span) BaggageItem(restrictedKey string) string {
	return ""
}

func (s Span) Tracer() opentracing.Tracer {
	return s.tracer
}

func (s Span) LogEvent(event string) {}

func (s Span) LogEventWithPayload(event string, payload interface{}) {}

func (s Span) Log(data opentracing.LogData) {}

func (s Span) OperationName() string {
	return s.operationName
}
