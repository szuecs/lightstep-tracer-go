package lightstep

import (
	"fmt"
	"log"
	"reflect"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
)

// Flushing Errors
const (
	ErrFlushingWhenClosed   = "LS Tracer cannot Flush, the tracer is closed."
	ErrFlushingWhenDisabled = "LS Tracer cannot Flush, the tracer is disabled."
)

/*
	Error Types
*/

type ClosedError interface {
	Closed()
	Error() string
}

type DisabledError interface {
	Disabled()
	Error() string
}

type DisconnectedError interface {
	Disconnected()
	Error() string
}

type DroppedSpansError interface {
	DroppedSpans() int
	Error() string
}

type NestedError interface {
	Nested() error
	Error() string
}

type nestedError struct {
	err error
}

func newNestedError(err error) NestedError {
	return nestedError{err: err}
}

func (e nestedError) Nested() error {
	return e.err
}

func (e nestedError) Error() string {
	return e.err.Error()
}

type closedError string

func newClosedError(msg string) ClosedError {
	return closedError(msg)
}

func (e closedError) Closed() {
}

func (e closedError) Error() string {
	return string(e)
}

type disabledError string

func newDisabledError(msg string) DisabledError {
	return disabledError(msg)
}

func (e disabledError) Disabled() {
}

func (e disabledError) Error() string {
	return string(e)
}

type disconnectedError nestedError

func newDisconnectedError(err error) DisconnectedError {
	return disconnectedError{err: err}
}

func (e disconnectedError) Disconnected() {
}

func (e disconnectedError) Nested() error {
	return e.err
}

func (e disconnectedError) Error() string {
	return e.err.Error()
}

type droppedSpansError struct {
	droppedSpans int
}

func newDroppedSpansError(droppedSpans int) DroppedSpansError {
	return &droppedSpansError{droppedSpans: droppedSpans}
}

func (err *droppedSpansError) DroppedSpans() int {
	return err.droppedSpans
}

func (err *droppedSpansError) Error() string {
	return fmt.Sprintf("client reported %d dropped spans", err.droppedSpans)
}

// newErrNotLisghtepTracer returns a typecasting error
func newErrNotLisghtepTracer(tracer opentracing.Tracer) error {
	return fmt.Errorf("Not a LightStep Tracer type: %v", reflect.TypeOf(tracer))
}

/*
	OnError Handlers
*/

// LogOnError logs errors using the standard go logger
func NewLogOnError() func(error) {
	return func(err error) {
		log.Println("LightStep error: ", err.Error())
	}
}

type logOneError struct {
	sync.Once
}

func (l *logOneError) OnError(err error) {
	l.Once.Do(func() {
		log.Printf("LightStep instrumentation error: (%s). NOTE: Set the Verbose option to enable more logging.\n", err.Error())
	})
}

// OnErrorLogOnce only logs the first error
func NewLogOnceOnError() func(error) {
	// Even if the flag is not set, always log at least one error.
	logger := logOneError{}
	return logger.OnError
}

// NewChannelOnError returns an OnError callback handler, and a channel that
// produces the errors. When the channel buffer is full, subsequent errors will
// be dropped. A buffer size of less than one is incorrect, and will be adjusted
// to a buffer size of one.
func NewChannelOnError(buffer int) (func(error), <-chan error) {
	if buffer < 1 {
		buffer = 1
	}

	errChan := make(chan error, buffer)

	handler := func(err error) {
		select {
		case errChan <- err:
		default:
		}
	}

	return handler, errChan
}
