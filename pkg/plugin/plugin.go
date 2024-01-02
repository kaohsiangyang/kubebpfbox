package plugin

import (
	"kubebpfbox/pkg/metric"
)

type Plugin interface {
	Name() string
	Gather(chan metric.Metric) error
}

var (
	Plugins map[string]Plugin
)

func Registry(name string, plugin Plugin) {
	Plugins[name] = plugin
}
