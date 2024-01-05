package plugin

import (
	"kubebpfbox/internal/metric"
)

type Plugin interface {
	Name() string
	Gather(chan metric.Metric) error
}

var (
	Plugins map[string]Plugin = make(map[string]Plugin)
)

func Registry(name string, plugin Plugin) {
	Plugins[name] = plugin
}
