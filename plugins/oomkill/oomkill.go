package oomkill

import (
	"kubebpfbox/global"
	"kubebpfbox/internal/metric"
	"kubebpfbox/internal/plugin"
)

type Oomkill struct{}

// Name returns the name of the plugin.
func (o *Oomkill) Name() string {
	return "oomkill"
}

// Gather collects metrics from the plugin.
func (o *Oomkill) Gather(c chan metric.Metric) error {
	global.Logger.Infof("Gather oomkill metrics")
	ch := make(chan Event)

	global.Logger.Infof("Load oomkill ebpf program")
	ebpf := NewOomkillEbpf(ch)
	if err := ebpf.Load(); err != nil {
		global.Logger.Errorf("Load oomkill ebpf program failed: %v", err)
		return err
	}
	defer ebpf.Unload()

	global.Logger.Infof("Start oomkill ebpf program")
	go func() {
		if err := ebpf.Start(); err != nil {
			global.Logger.Errorf("Start oomkill ebpf program failed: %v", err)
		}
	}()

	for m := range ch {
		global.Logger.Infof("Get oomkill event: %+v", m.String())
		c <- m.CovertMetric()
	}
	return nil
}

func init() {
	plugin.Registry("oomkill", &Oomkill{})
}
