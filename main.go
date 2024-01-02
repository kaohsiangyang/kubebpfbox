package main

import (
	"kubebpfbox/global"
	"kubebpfbox/pkg/ip2pod"
	"kubebpfbox/pkg/k8s"
)

func init() {
	setupK8s()
}

func setupK8s() {
	global.PodController = *k8s.NewPodController()
	global.Ip2Pod = *ip2pod.NewIP2Pod(global.PodController)
	global.Ip2Pod.Registry()
	go global.PodController.Run()
}

func main() {

}
