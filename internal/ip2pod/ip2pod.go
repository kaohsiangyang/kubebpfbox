package ip2pod

import (
	"kubebpfbox/global"
	"kubebpfbox/internal/k8s"
	"net"
	"sync"

	core "k8s.io/api/core/v1"
)

type IP2Pod struct {
	podController *k8s.PodController
	Pods          map[string]*core.Pod
}

var ip2pod *IP2Pod
var once sync.Once

// GetIP2Pod returns a singleton IP2Pod
func GetIP2Pod() *IP2Pod {
	once.Do(func() {
		ip2pod = &IP2Pod{
			podController: k8s.GetPodController(),
			Pods:          make(map[string]*core.Pod),
		}
	})
	return ip2pod
}

// GetPodByIP returns a pod by ip
func (i *IP2Pod) GetPodByIP(ip string) (pod *core.Pod, ok bool) {
	if pod, ok := i.Pods[ip]; ok {
		return pod, true
	}
	return nil, false
}

func addPod(newPod *core.Pod, nodename string) {
	if newPod.Spec.NodeName != nodename {
		global.Logger.Warnf("Pod %s is not in node %s", newPod.Name, nodename)
		return
	}
	podIP := newPod.Status.PodIP
	ip := net.ParseIP(podIP)

	GetIP2Pod().Pods[ip.String()] = newPod
	global.Logger.Infof("Add Pod %s with IP %s successfully", newPod.Name, podIP)
}

func deletePod(oldPod *core.Pod, nodename string) {
	if oldPod.Spec.NodeName != nodename {
		global.Logger.Warnf("Pod %s is not in node %s", oldPod.Name, nodename)
		return
	}
	podIP := oldPod.Status.PodIP
	ip := net.ParseIP(podIP)
	delete(GetIP2Pod().Pods, ip.String())
	global.Logger.Infof("Delete Pod %s with IP %s successfully", oldPod.Name, podIP)
}

func updatePod(oldPod *core.Pod, newPod *core.Pod, nodename string) {
	if newPod.Spec.NodeName != nodename {
		global.Logger.Warnf("Pod %s is not in node %s", newPod.Name, nodename)
		return
	}
	podIP := newPod.Status.PodIP
	ip := net.ParseIP(podIP)
	GetIP2Pod().Pods[ip.String()] = newPod
	global.Logger.Infof("Update Pod %s with IP %s successfully", newPod.Name, podIP)
}

// Registry registers IP2Pod to pod controller
func (i *IP2Pod) Registry() {
	i.podController.AddEventHandlers = append(i.podController.AddEventHandlers, addPod)
	i.podController.DeleteEventHandlers = append(i.podController.DeleteEventHandlers, deletePod)
	i.podController.UpdateEventHandlers = append(i.podController.UpdateEventHandlers, updatePod)
	global.Logger.Info("Registry IP2Pod to pod controller successfully")
}
