package lightstep

import "fmt"

var (
	errConnectionWasClosed = fmt.Errorf("the connection was closed")
	errTracerDisabled      = fmt.Errorf("tracer is disabled; aborting Flush()")
)
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
