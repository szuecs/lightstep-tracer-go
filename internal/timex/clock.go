package timex

import "time"

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
	Since(time.Time) time.Duration
	NewTicker(time.Duration) Ticker
}

func NewClock() Clock {
	return clock{}
}

type clock struct{}

func (c clock) Now() time.Time {
	return time.Now()
}

func (c clock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (c clock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c clock) NewTicker(d time.Duration) Ticker {
	return newTicker(d)
}
