package internal

type SpanOption func(*spanConfig)

func WithRecorder(recorder RecordSpan) SpanOption {
	return func(c *spanConfig) {
		c.recorder = recorder
	}
}

func defaultSpanConfig() *spanConfig {
	return &spanConfig{
		recorder: noopRecordSpan,
	}
}

type spanConfig struct {
	recorder RecordSpan
}

type RecordSpan func(span Span)

func noopRecordSpan(span Span) {}
