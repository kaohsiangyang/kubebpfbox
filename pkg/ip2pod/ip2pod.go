package ip2pod

import (
	"kubebpfbox/pkg/k8s"
	"net"

	core "k8s.io/api/core/v1"
)

type IP2Pod struct {
	podController k8s.PodController
	Pods          map[*net.IP]*core.Pod
}

func NewIP2Pod(podController k8s.PodController) *IP2Pod {
	return &IP2Pod{
		podController: podController,
		Pods:          make(map[*net.IP]*core.Pod),
	}
}

func (i *IP2Pod) GetPodByIP(ip net.IP) (pod *core.Pod, ok bool) {
	if pod, ok := i.Pods[&ip]; ok {
		return pod, true
	}
	return nil, false
}

func (i *IP2Pod) addPod(newPod *core.Pod, nodename string) {
	if newPod.Spec.NodeName != nodename {
		return
	}
	podIP := newPod.Status.PodIP
	ip := net.ParseIP(podIP)
	i.Pods[&ip] = newPod
}

func (i *IP2Pod) deletePod(oldPod *core.Pod, nodename string) {
	if oldPod.Spec.NodeName != nodename {
		return
	}
	podIP := oldPod.Status.PodIP
	ip := net.ParseIP(podIP)
	delete(i.Pods, &ip)
}

func (i *IP2Pod) updatePod(oldPod *core.Pod, newPod *core.Pod, nodename string) {
	if newPod.Spec.NodeName != nodename {
		return
	}
	podIP := newPod.Status.PodIP
	ip := net.ParseIP(podIP)
	i.Pods[&ip] = newPod
}

func (i *IP2Pod) Registry() {
	i.podController.AddEventHandlers = append(i.podController.AddEventHandlers, i.addPod)
	i.podController.DeleteEventHandlers = append(i.podController.DeleteEventHandlers, i.deletePod)
	i.podController.UpdateEventHandlers = append(i.podController.UpdateEventHandlers, i.updatePod)
}
