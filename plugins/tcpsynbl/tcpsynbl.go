package tcpsynbl

import (
	"kubebpfbox/global"
	"kubebpfbox/internal/metric"
	"kubebpfbox/internal/plugin"
)

type Tcpsynbl struct{}

// Name returns the name of the plugin.
func (t *Tcpsynbl) Name() string {
	return "tcpsynbl"
}

// Gather collects metrics from the plugin.
func (t *Tcpsynbl) Gather(c chan metric.Metric) error {
	global.Logger.Infof("Gather tcpsynbl metrics")
	ch := make(chan Event)

	global.Logger.Infof("Load tcpsynbl ebpf program")
	ebpf := NewTcpsynblEbpf(ch)
	if err := ebpf.Load(); err != nil {
		global.Logger.Errorf("Load tcpsynbl ebpf program failed: %v", err)
		return err
	}
	defer ebpf.Unload()

	global.Logger.Infof("Start tcpsynbl ebpf program")
	go func() {
		if err := ebpf.Start(); err != nil {
			global.Logger.Errorf("Start tcpsynbl ebpf program failed: %v", err)
		}
	}()

	for m := range ch {
		global.Logger.Infof("Get tcpsynbl event: %+v", m.String())
		c <- m.CovertMetric()
	}
	return nil
}

func init() {
	plugin.Registry("tcpsynbl", &Tcpsynbl{})
}
