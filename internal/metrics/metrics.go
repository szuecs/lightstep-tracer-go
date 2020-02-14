package metrics

import (
	"context"

	"github.com/shirou/gopsutil/cpu"
)

type Metrics struct {
	CPU map[string]CPU
}

type CPU struct {
	User   float64
	System float64
	Idle   float64
	Steal  float64
	Nice   float64
}

func Measure(ctx context.Context) (Metrics, error) {
	cpum, err := cpu.TimesWithContext(ctx, true)
	if err != nil {
		return Metrics{}, err
	}

	metrics := Metrics{
		CPU: make(map[string]CPU),
	}

	for _, m := range cpum {
		metrics.CPU[m.CPU] = CPU{
			User:   m.User,
			System: m.System,
			Idle:   m.Idle,
			Steal:  m.Steal,
			Nice:   m.Nice,
		}
	}

	return metrics, nil
}
