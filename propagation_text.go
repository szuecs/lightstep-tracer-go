package lightstep

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
)

const (
	prefixTracerState = "ot-tracer-"
	prefixBaggage     = "ot-baggage-"

	tracerStateFieldCount = 3
	fieldNameTraceID      = prefixTracerState + "traceid"
	fieldNameSpanID       = prefixTracerState + "spanid"
	fieldNameSampled      = prefixTracerState + "sampled"

	vendorKey = "lightstep"

	traceParentKey = "traceparent"
	traceStateKey  = "tracestate"

	maxTraceStateLen = 512
)

var (
	theTextMapPropagator textMapPropagator

	traceParentRegexp     *regexp.Regexp
	traceParentRegexpOnce sync.Once

	traceStateRegexp     *regexp.Regexp
	traceStateRegexpOnce sync.Once
)

type textMapPropagator struct{}

func (textMapPropagator) Inject(
	spanContext opentracing.SpanContext,
	opaqueCarrier interface{},
) error {
	sc, ok := spanContext.(SpanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	carrier, ok := opaqueCarrier.(opentracing.TextMapWriter)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}
	leadingTraceID := strconv.FormatUint(sc.LeadingTraceID, 16)
	traceID := strconv.FormatUint(sc.TraceID, 16)
	spanID := strconv.FormatUint(sc.SpanID, 16)

	carrier.Set(fieldNameTraceID, traceID)
	carrier.Set(fieldNameSpanID, spanID)
	carrier.Set(fieldNameSampled, "true")
	carrier.Set(traceParentKey, fmt.Sprintf("00-%016s%016s-%016s-01", leadingTraceID, traceID, spanID))

	var baggage []string
	for k, v := range sc.Baggage {
		carrier.Set(prefixBaggage+k, v)
		baggage = append(baggage, fmt.Sprintf("%s=%s", k, v))
	}
	encodedBaggage := base64.RawURLEncoding.EncodeToString([]byte(strings.Join(baggage, ",")))
	traceState := fmt.Sprintf("%s=%s", vendorKey, encodedBaggage)

	traceStateLen := len(traceState)

	for _, ts := range sc.TraceState {
		encodedTS := fmt.Sprintf(",%s=%s", ts.Vendor, ts.Value)
		traceStateLen += len(encodedTS)

		if traceStateLen > maxTraceStateLen {
			break
		}

		traceState += encodedTS
	}

	carrier.Set(traceStateKey, traceState)

	return nil
}

func (textMapPropagator) Extract(
	opaqueCarrier interface{},
) (opentracing.SpanContext, error) {
	carrier, ok := opaqueCarrier.(opentracing.TextMapReader)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}

	requiredFieldCount := 0
	var traceID, leadingTraceID, spanID uint64
	var opaqueTraceState []OpaqueTraceState
	var err error
	decodedBaggage := map[string]string{}
	err = carrier.ForeachKey(func(k, v string) error {
		switch strings.ToLower(k) {
		case fieldNameTraceID:
			traceID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
			requiredFieldCount++
		case fieldNameSpanID:
			spanID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
			requiredFieldCount++
		case fieldNameSampled:
			requiredFieldCount++
		case traceParentKey:
			traceParentRegexpOnce.Do(compileTraceParentRegexp)
			if traceParentRegexp != nil {
				matches := traceParentRegexp.FindAllStringSubmatch(v, -1)
				if len(matches) == 1 && len(matches[0]) == 4 {
					requiredFieldCount += 3 // traceparent regex checks for trace ID, span ID, and sampled bit

					leadingTraceID, err = strconv.ParseUint(matches[0][1], 16, 64)
					if err != nil {
						return opentracing.ErrSpanContextCorrupted
					}
					traceID, err = strconv.ParseUint(matches[0][2], 16, 64)
					if err != nil {
						return opentracing.ErrSpanContextCorrupted
					}
					spanID, err = strconv.ParseUint(matches[0][3], 16, 64)
					if err != nil {
						return opentracing.ErrSpanContextCorrupted
					}
				}
			}
		case traceStateKey:
			traceStateRegexpOnce.Do(compileTraceStateRegexp)

			if traceStateRegexp != nil {
				traceState := strings.Split(v, ",")
				for _, ts := range traceState {
					matches := traceStateRegexp.FindAllStringSubmatch(ts, -1)
					if len(matches) == 1 && len(matches[0]) == 3 {
						tsVendor := matches[0][1]
						tsValue := matches[0][2]
						if tsVendor == vendorKey {
							var dBaggage []byte
							dBaggage, err = base64.RawURLEncoding.DecodeString(tsValue)
							if err != nil {
								return opentracing.ErrSpanContextCorrupted
							}

							baggage := strings.Split(string(dBaggage), ",")
							for _, item := range baggage {
								splitBaggage := strings.SplitN(item, "=", 2)
								if len(splitBaggage) == 2 {
									decodedBaggage[splitBaggage[0]] = splitBaggage[1]
								}
							}
						} else {
							opaqueTraceState = append(opaqueTraceState, OpaqueTraceState{
								Vendor: tsVendor,
								Value:  tsValue,
							})
						}
					}
				}
			}
		default:
			lowercaseK := strings.ToLower(k)
			if strings.HasPrefix(lowercaseK, prefixBaggage) {
				decodedBaggage[strings.TrimPrefix(lowercaseK, prefixBaggage)] = v
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if requiredFieldCount < tracerStateFieldCount {
		if requiredFieldCount == 0 {
			return nil, opentracing.ErrSpanContextNotFound
		}
		return nil, opentracing.ErrSpanContextCorrupted
	}

	return SpanContext{
		TraceID:        traceID,
		LeadingTraceID: leadingTraceID,
		SpanID:         spanID,
		Baggage:        decodedBaggage,
		TraceState:     opaqueTraceState,
	}, nil
}

func compileTraceParentRegexp() {
	traceParentRegexp, _ = regexp.Compile(`^[[:xdigit:]]{2}-([[:xdigit:]]{16})([[:xdigit:]]{16})-([[:xdigit:]]{16})-[[:xdigit:]]{2}$`)
}

func compileTraceStateRegexp() {
	traceStateRegexp, _ = regexp.Compile(`^\s*([a-z0-9_\-/]+)=([\x21-\x2b\x2d-\x3c\x3e-\x7e]*)\s*$`)
}
