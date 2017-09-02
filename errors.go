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
