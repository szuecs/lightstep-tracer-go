package internal

type SpanContext struct {
	traceID uint64
	spanID  uint64
	baggage map[string]string
}

func (sc SpanContext) ForeachBaggageItem(handler func(key, value string) bool) {}
