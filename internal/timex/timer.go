package timex

import "time"

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type ticker struct {
	t *time.Ticker
}

func newTicker(d time.Duration) Ticker {
	return &ticker{
		t: time.NewTicker(d),
	}
}

func (t *ticker) C() <-chan time.Time {
	return t.t.C
}

func (t ticker) Stop() {
	t.t.Stop()
}
