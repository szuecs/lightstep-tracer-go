package testtimex

import (
	"sync"
	"time"

	"github.com/lightstep/lightstep-tracer-go/internal/timex"
)

func NewClock(start time.Time) timex.Clock {
	return &clock{
		now:  start,
		lock: &sync.Mutex{},
	}
}

type clock struct {
	now     time.Time
	lock    *sync.Mutex
	tickers []*ticker
}

func (c *clock) Now() time.Time {
	return c.now
}

func (c *clock) Sleep(d time.Duration) {
	if d > 0 {
		c.lock.Lock()
		defer c.lock.Unlock()

		c.now = c.now.Add(d)

		for _, t := range c.tickers {
			go t.send(c.now)
		}
	}
}

func (c *clock) Since(t time.Time) time.Duration {
	return c.now.Sub(t)
}

func (c *clock) NewTicker(d time.Duration) timex.Ticker {
	if d <= 0 {
		panic("duration must be > 0")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	ticker := &ticker{
		lock: &sync.Mutex{},
		c:    make(chan time.Time),
		d:    d,
		next: c.now.Add(d),
	}
	c.tickers = append(c.tickers, ticker)

	return ticker
}

type ticker struct {
	lock   *sync.Mutex
	c      chan time.Time
	d      time.Duration
	next   time.Time
	closed bool
}

func (t *ticker) C() <-chan time.Time {
	return t.c
}

func (t *ticker) Stop() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if !t.closed {
		t.closed = true
		close(t.c)
	}
}

func (t *ticker) send(now time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if !t.closed {
		if t.next.Before(now) || t.next.Equal(now) {
			t.next = t.next.Add(t.d)

			t.c <- now
		}
	}
}
