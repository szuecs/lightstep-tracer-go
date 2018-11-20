package lightstep

import (
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

// RawSpan encapsulates all state associated with a (finished) LightStep Span.
type RawSpan struct {
	// Those recording the RawSpan should also record the contents of its
	// SpanContext.
	Context SpanContext

	// The SpanID of this SpanContext's first intra-trace reference (i.e.,
	// "parent"), or 0 if there is no parent.
	ParentSpanID uint64

	// The name of the "operation" this span is an instance of. (Called a "span
	// name" in some implementations)
	Operation string

	// We store <start, duration> rather than <start, end> so that only
	// one of the timestamps has global clock uncertainty issues.
	Start    time.Time
	Duration time.Duration

	// Essentially an extension mechanism. Can be used for many purposes,
	// not to be enumerated here.
	Tags opentracing.Tags

	// The span's "microlog".
	Logs []opentracing.LogRecord
}

// SpanContext holds lightstep-specific Span metadata.
type SpanContext struct {
	// A probabilistically unique identifier for a [multi-span] trace.
	TraceID uint64

	// A probabilistically unique identifier for a span.
	SpanID uint64

	// The span's associated baggage.
	Baggage map[string]string // initialized on first use

	// Data propagated across vendors.
	TraceState []OpaqueTraceState

	// Used to store the leading 16 bytes of a 32-byte trace ID
	// so that the full ID may be propagated across vendors,
	// i.e., if there is a 32-byte trace ID in the `traceparent` header
	LeadingTraceID uint64
}

// OpaqueTraceState contains data from other vendors, propagated via the `tracestate` header
type OpaqueTraceState struct {
	Vendor string
	Value  string
}

// ForeachBaggageItem belongs to the opentracing.SpanContext interface
func (c SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	for k, v := range c.Baggage {
		if !handler(k, v) {
			break
		}
	}
}

// WithBaggageItem returns an entirely new basictracer SpanContext with the
// given key:value baggage pair set.
func (c SpanContext) WithBaggageItem(key, val string) SpanContext {
	var newBaggage map[string]string
	if c.Baggage == nil {
		newBaggage = map[string]string{key: val}
	} else {
		newBaggage = make(map[string]string, len(c.Baggage)+1)
		for k, v := range c.Baggage {
			newBaggage[k] = v
		}
		newBaggage[key] = val
	}
	// Use positional parameters so the compiler will help catch new fields.
	return SpanContext{c.TraceID, c.SpanID, newBaggage, c.TraceState, c.LeadingTraceID}
}
