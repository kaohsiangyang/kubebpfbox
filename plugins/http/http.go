package http

import (
	"kubebpfbox/global"
	"kubebpfbox/internal/metric"
	"kubebpfbox/internal/plugin"
	"strconv"
	"time"
)

type Http struct{}

// Name returns the name of the plugin.
func (h *Http) Name() string {
	return "http"
}

// Gather collects metrics from the plugin.
func (h *Http) Gather(c chan metric.Metric) error {
	global.Logger.Infof("Gather http metrics")
	ch := make(chan Traffic)

	global.Logger.Infof("Load http ebpf program")
	ebpf := NewHttpEbpf(ch)
	if err := ebpf.Load(); err != nil {
		global.Logger.Errorf("Load http ebpf program failed: %v", err)
		return err
	}
	defer ebpf.Unload()

	global.Logger.Infof("Start http ebpf program")
	go func() {
		if err := ebpf.Start(); err != nil {
			global.Logger.Errorf("Start http ebpf program failed: %v", err)
		}
	}()

	redMetric := make(map[string]RED)
	calTicker := time.NewTicker(60 * time.Second)
	for {
		select {
		case m := <-ch:
			isErr := 0
			codeNum, _ := strconv.Atoi(m.Code)
			if codeNum > 499 {
				isErr = 1
			}
			if v, ok := redMetric[m.PodName]; ok {
				v.RequestCount += 1
				v.ErrCount += isErr
				v.DurationCount += int(m.Duration)
				redMetric[m.PodName] = v
			} else {
				redMetric[m.PodName] = RED{
					PodName:       m.PodName,
					NodeName:      m.NodeName,
					NameSpace:     m.NameSpace,
					ServiceName:   m.NameSpace,
					RequestCount:  1,
					ErrCount:      isErr,
					DurationCount: int(m.Duration),
				}
			}
			global.Logger.Debugf("Get traffic metric: %s", m.String())
			c <- m.CovertMetric()
		case <-calTicker.C:
			for k, v := range redMetric {
				v.QPS = float32(v.RequestCount) / 60
				v.ErrRate = float32(v.ErrCount) / float32(v.RequestCount) * 100
				v.Duration = float32(v.DurationCount) / float32(v.RequestCount)
				global.Logger.Debugf("Get red metric: %s", v.String())
				c <- v.CovertMetric()
				delete(redMetric, k)
			}
		}
	}
}

func init() {
	plugin.Registry("http", &Http{})
}
