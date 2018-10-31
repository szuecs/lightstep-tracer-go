package testtimex_test

import (
	"time"

	"github.com/lightstep/lightstep-tracer-go/internal/timex"
	. "github.com/lightstep/lightstep-tracer-go/internal/timex/testtimex"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Clock", func() {
	durations := []time.Duration{0, time.Minute, time.Hour}

	var (
		start time.Time

		subject timex.Clock
	)

	BeforeEach(func() {
		start = time.Now()
		subject = NewClock(start)
	})

	Describe("#Now", func() {
		It("starts out at the provided time", func() {
			Expect(subject.Now()).To(Equal(start))
		})
	})

	Describe("#Sleep", func() {
		It("advances the current time by the provided duration", func() {
			expectedTime := start
			for _, d := range durations {
				expectedTime = expectedTime.Add(d)
				subject.Sleep(d)
				Expect(subject.Now()).To(Equal(expectedTime))
			}
		})

		It("does nothing if provided 0", func() {
			subject.Sleep(0)
			Expect(subject.Now()).To(Equal(start))
		})

		It("does nothing if provided a negative duration", func() {
			subject.Sleep(time.Hour * -1)
			Expect(subject.Now()).To(Equal(start))
		})

		It("can be called concurrently", func() {
			done := make(chan interface{})
			expectedTime := start

			for _, d := range durations {
				expectedTime = expectedTime.Add(d)

				go func(dur time.Duration) {
					subject.Sleep(dur)
					done <- nil
				}(d)
			}

			for i := 0; i < len(durations); i++ {
				<-done
			}

			Expect(subject.Now()).To(Equal(expectedTime))
		})
	})

	Describe("#Since", func() {
		It("provides the time elapsed since provided time", func() {
			for _, d := range durations {
				t := start.Add(d)
				Expect(subject.Since(t)).To(Equal(start.Sub(t)))
			}
		})
	})

	Describe("#NewTicker", func() {
		It("panics when given an invalid duration", func() {
			Expect(func() { subject.NewTicker(0) }).To(Panic())
			Expect(func() { subject.NewTicker(time.Hour * -1) }).To(Panic())
		})

		It("can handle multiple tickers", func() {
			d1 := time.Minute
			t1 := subject.NewTicker(d1)

			d2 := time.Hour
			t2 := subject.NewTicker(d2)

			subject.Sleep(d1)

			Eventually(t1.C()).Should(Receive())
			Consistently(t2.C()).ShouldNot(Receive())

			subject.Sleep(d2)

			Eventually(t1.C()).Should(Receive())
			Eventually(t2.C()).Should(Receive())
		})

		It("returns tickers that can be stopped independently", func() {
			d := time.Minute

			t1 := subject.NewTicker(d)
			t2 := subject.NewTicker(d)

			t1.Stop()

			subject.Sleep(d * 2)

			Consistently(t1.C()).Should(BeClosed())
			Eventually(t2.C()).Should(Receive())
		})
	})
})
