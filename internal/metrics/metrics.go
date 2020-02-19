package metrics

import (
	"context"
	"os"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
)

type Metrics struct {
	ProcessCPU ProcessCPU
	CPU        map[string]CPU
}

type ProcessCPU struct {
	User   float64
	System float64
}

type CPU struct {
	User   float64
	System float64
	Idle   float64
	Steal  float64
	Nice   float64
}

func Measure(ctx context.Context) (Metrics, error) {
	p, err := process.NewProcess(int32(os.Getpid())) // TODO: cache the process
	if err != nil {
		return Metrics{}, err
	}

	processTimes, err := p.TimesWithContext(ctx) // returns user and system time for process
	if err != nil {
		return Metrics{}, err
	}

	systemTimes, err := cpu.TimesWithContext(ctx, true)
	if err != nil {
		return Metrics{}, err
	}

	metrics := Metrics{
		ProcessCPU: ProcessCPU{
			User:   processTimes.User,
			System: processTimes.System,
		},
		CPU: make(map[string]CPU),
	}

	for _, m := range systemTimes {
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
