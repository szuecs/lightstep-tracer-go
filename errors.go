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

type ErrClosed interface {
	Closed()
	error
}

type ErrDisabled interface {
	Disabled()
	error
}

type ErrDisconected interface {
	Disconnected()
	error
}

type ErrDroppedSpans interface {
	DroppedSpans() int
	error
}

type ErrNested interface {
	Nested() error
	error
}

type nestedError struct {
	err error
}

func newErrNested(err error) ErrNested {
	return nestedError{err: err}
}

func (e nestedError) Nested() error {
	return e.err
}

func (e nestedError) Error() string {
	return e.err.Error()
}

type closedError string

func newErrClosed(msg string) ErrClosed {
	return closedError(msg)
}

func (e closedError) Closed() {
}

func (e closedError) Error() string {
	return string(e)
}

type disabledError string

func newErrDisabled(msg string) ErrDisabled {
	return disabledError(msg)
}

func (e disabledError) Disabled() {
}

func (e disabledError) Error() string {
	return string(e)
}

type disconnectedError nestedError

func newErrDisconected(err error) ErrDisconected {
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

func newErrDroppedSpans(droppedSpans int) ErrDroppedSpans {
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
