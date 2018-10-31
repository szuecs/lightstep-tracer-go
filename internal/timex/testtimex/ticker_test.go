package testtimex_test

import (
	"time"

	"github.com/lightstep/lightstep-tracer-go/internal/timex"
	. "github.com/lightstep/lightstep-tracer-go/internal/timex/testtimex"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ticker", func() {
	var (
		start time.Time
		clock timex.Clock
	)

	BeforeEach(func() {
		start = time.Now()
		clock = NewClock(start)
	})

	Describe("#C", func() {
		It("fires only when expected", func() {
			d := time.Minute
			subject := clock.NewTicker(d * 2)

			Consistently(subject.C()).ShouldNot(Receive())

			expectedTime := start
			for i := 0; i < 3; i++ {
				clock.Sleep(d)
				expectedTime = expectedTime.Add(d)

				Consistently(subject.C()).ShouldNot(Receive())

				clock.Sleep(d)
				expectedTime = expectedTime.Add(d)

				Eventually(subject.C()).Should(Receive(Equal(expectedTime)))
			}
		})

		It("may drop fires", func() {
			d := time.Minute
			subject := clock.NewTicker(d)

			clock.Sleep(d * 5)

			Eventually(subject.C()).Should(Receive())
			Consistently(subject.C()).ShouldNot(Receive())
		})
	})

	Describe("#Stop", func() {
		It("closes the channel", func() {
			d := time.Minute
			subject := clock.NewTicker(d)

			subject.Stop()

			for i := 0; i < 5; i++ {
				clock.Sleep(d)
			}

			Consistently(subject.C()).Should(BeClosed())
		})

		It("can be called multiple times", func() {
			subject := clock.NewTicker(time.Minute)

			Expect(func() {
				subject.Stop()
				subject.Stop()
			}).NotTo(Panic())
		})
	})
})
