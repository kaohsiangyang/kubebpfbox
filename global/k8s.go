package global

import (
	"kubebpfbox/pkg/ip2pod"
	"kubebpfbox/pkg/k8s"
)

var (
	PodController k8s.PodController
	Ip2Pod        ip2pod.IP2Pod
)
