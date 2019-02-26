package randx

import (
	"runtime"
	"time"

	"github.com/lightstep/lightstep-tracer-go/lightstep/rand"
)

var (
	// create a random pool with size equal to 16 generators or number of CPU Cores which ever is higher to spread
	// random int call loads across multiple go routines. This number is obtained via local benchmarking
	// where any number more than 16 reaches a point of diminishing return given the test scenario.
	randomPool = rand.NewPool(time.Now().UnixNano(), uint64(max(16, runtime.NumCPU())))
)

func GenSeededGUID(opts ...Option) uint64 {
	c := defaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	return uint64(c.randomPool.Pick().Int63())
}

func GenSeededGUID2(opts ...Option) (uint64, uint64) {
	c := defaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	n1, n2 := c.randomPool.Pick().TwoInt63()
	return uint64(n1), uint64(n2)
}

type Option func(*config)

func WithRandomPool(randomPool *rand.Pool) Option {
	return func(c *config) {
		c.randomPool = randomPool
	}
}

type config struct {
	randomPool *rand.Pool
}

func defaultConfig() *config {
	return &config{
		randomPool: randomPool,
	}
}

// max returns the larger value among a and b
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
