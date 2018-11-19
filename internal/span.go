package internal

import (
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type Span struct {
	// lock *sync.RWMutex
	operationName string
	ctx           SpanContext
	tracer        opentracing.Tracer
	recorder      RecordSpan
	finishOnce    *sync.Once
	tags          map[string]interface{}
	logs          []opentracing.LogRecord
}

func NewSpan(operationName string, opts ...SpanOption) Span {
	c := defaultSpanConfig()
	for _, opt := range opts {
		opt(c)
	}

	ctx := SpanContext{
		baggage: make(map[string]string),
	}

	return Span{
		// lock: &sync.RWMutex{},
		operationName: operationName,
		ctx:           ctx,
		recorder:      c.recorder,
		finishOnce:    &sync.Once{},
		tags:          make(map[string]interface{}),
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
	s.tags[key] = value
	return s
}

func (s Span) LogFields(fields ...log.Field) {
	s.log(opentracing.LogRecord{
		Fields: fields,
	})
}

func (s Span) LogKV(alternatingKeyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(alternatingKeyValues...)
	if err != nil {
		fields = []log.Field{log.Error(err), log.String("function", "LogKV")}
	}

	s.log(opentracing.LogRecord{Fields: fields})
}

func (s Span) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	s.ctx.baggage[restrictedKey] = value
	return s
}

func (s Span) BaggageItem(restrictedKey string) string {
	return s.ctx.baggage[restrictedKey]
}

func (s Span) Tracer() opentracing.Tracer {
	return s.tracer
}

func (s Span) LogEvent(event string) {
	s.LogEventWithPayload(event, nil)
}

func (s Span) LogEventWithPayload(event string, payload interface{}) {
	s.Log(opentracing.LogData{
		Event:   event,
		Payload: payload,
	})
}

func (s Span) Log(data opentracing.LogData) {
	s.log(data.ToLogRecord())
}

func (s Span) OperationName() string {
	return s.operationName
}

func (s Span) log(lr opentracing.LogRecord) {
	if lr.Timestamp.IsZero() {
		lr.Timestamp = time.Now()
	}

	s.logs = append(s.logs, lr)
}
