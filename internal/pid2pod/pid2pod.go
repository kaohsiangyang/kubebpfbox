package pid2pod

import (
	"bufio"
	"fmt"
	"kubebpfbox/global"
	"kubebpfbox/internal/k8s"
	"os"
	"regexp"
	"sync"

	core "k8s.io/api/core/v1"
)

type PodInfo struct {
	Namespace     string
	NodeName      string
	PodName       string
	PodUID        string
	PodLabel      map[string]string
	ContainerID   string
	ContainerName string
}

type Pid2Pod struct {
	PodController *k8s.PodController
	Pods          map[string]*core.Pod
}

var pid2Pod *Pid2Pod
var once sync.Once

func GetPid2Pod() *Pid2Pod {
	once.Do(func() {
		pid2Pod = &Pid2Pod{
			PodController: k8s.GetPodController(),
			Pods:          make(map[string]*core.Pod),
		}
	})
	return pid2Pod
}

func (p *Pid2Pod) GetPodInfoByPid(pid uint32) (podInfo *PodInfo, err error) {
	podId, containerId, err := lookupIdByPid(pid)
	if err != nil {
		return nil, err
	}
	if pod, ok := p.Pods[podId]; ok {
		var containerName string
		for _, Container := range pod.Status.ContainerStatuses {
			if Container.ContainerID == containerId {
				containerName = Container.Name
			}
		}
		if containerName == "" {
			global.Logger.Errorf("no container found for pid %d", pid)
			return nil, fmt.Errorf("no container found for pid %d", pid)
		}
		return &PodInfo{
			Namespace:     pod.Namespace,
			NodeName:      pod.Spec.NodeName,
			PodName:       pod.Name,
			PodUID:        string(pod.UID),
			PodLabel:      pod.Labels,
			ContainerID:   containerId,
			ContainerName: containerName,
		}, nil
	}
	return nil, fmt.Errorf("no pod found for pid %d", pid)
}

func addPod(newPod *core.Pod, nodename string) {
	if newPod.Spec.NodeName != nodename {
		global.Logger.Warnf("Pod %s is not in node %s", newPod.Name, nodename)
		return
	}
	GetPid2Pod().Pods[string(newPod.UID)] = newPod
	global.Logger.Infof("Add Pod %s successfully", newPod.Name)
}

func deletePod(oldPod *core.Pod, nodename string) {
	if oldPod.Spec.NodeName != nodename {
		global.Logger.Warnf("Pod %s is not in node %s", oldPod.Name, nodename)
		return
	}
	delete(GetPid2Pod().Pods, string(oldPod.UID))
	global.Logger.Infof("Delete Pod %s successfully", oldPod.Name)
}

func updatePod(oldPod *core.Pod, newPod *core.Pod, nodename string) {
	if newPod.Spec.NodeName != nodename {
		global.Logger.Warnf("Pod %s is not in node %s", newPod.Name, nodename)
		return
	}
	GetPid2Pod().Pods[string(newPod.UID)] = newPod
	global.Logger.Infof("Update Pod %s successfully", newPod.Name)
}

func (p *Pid2Pod) Registry() {
	p.PodController.AddEventHandlers = append(p.PodController.AddEventHandlers, addPod)
	p.PodController.DeleteEventHandlers = append(p.PodController.DeleteEventHandlers, deletePod)
	p.PodController.UpdateEventHandlers = append(p.PodController.UpdateEventHandlers, updatePod)
	global.Logger.Info("Registry Pid2Pod to pod controller successfully")
}

var (
	kubeSystemdPattern = regexp.MustCompile(
		`\d+:.+:/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod([0-9a-f_]{36}).slice/cri-containerd-([0-9a-f_]{64}).scope`)
)

func lookupIdByPid(pid uint32) (podId string, containerId string, err error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		global.Logger.Debug("line: ", line)
		if parts := kubeSystemdPattern.FindStringSubmatch(line); len(parts) == 3 {
			global.Logger.Debug("parts: ", parts)
			return parts[1], parts[2], nil
		}
	}
	return "", "", fmt.Errorf("no pod found for pid %d", pid)
}
