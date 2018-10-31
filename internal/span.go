package internal

import (
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type Span struct {
	operationName string
	ctx           SpanContext
	tracer        opentracing.Tracer
}

func NewSpan(operationName string) Span {
	return Span{
		operationName: operationName,
	}
}

func (s Span) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

func (s Span) FinishWithOptions(opts opentracing.FinishOptions) {}

func (s Span) Context() opentracing.SpanContext {
	return s.ctx
}

func (s Span) SetOperationName(operationName string) opentracing.Span {
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
