package internal

type SpanContext struct {
	TraceID uint64
	SpanID  uint64
	Baggage map[string]string
}

func (sc SpanContext) ForeachBaggageItem(handler func(key, value string) bool) {}
