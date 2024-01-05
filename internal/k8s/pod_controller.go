package k8s

import (
	"kubebpfbox/global"
	"os"
	"sync"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type PodController struct {
	stopper             chan struct{}
	informer            cache.SharedIndexInformer
	AddEventHandlers    []func(newPod *core.Pod, nodename string)
	DeleteEventHandlers []func(oldPod *core.Pod, nodename string)
	UpdateEventHandlers []func(oldPod *core.Pod, newPod *core.Pod, nodename string)
}

var podController *PodController
var once sync.Once

func GetPodController() *PodController {
	once.Do(func() {
		factory := NewInformerFactory().Factory

		informer := factory.Core().V1().Pods().Informer()

		podController = &PodController{
			stopper:  make(chan struct{}),
			informer: informer,
		}
	})
	return podController
}

// Run starts the pod controller
func (p *PodController) Run() {
	nodename := os.Getenv("NODE_NAME")
	if nodename == "" {
		global.Logger.Fatalf("Environment variables 'NODE_NAME' need to set !")
	}

	p.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*core.Pod)
			for _, handler := range p.AddEventHandlers {
				handler(pod, nodename)
			}
			global.Logger.Infof("Add Pod %s successfully", pod.Name)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*core.Pod)
			for _, handler := range p.DeleteEventHandlers {
				handler(pod, nodename)
			}
			global.Logger.Infof("Delete Pod %s successfully", pod.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*core.Pod)
			newPod := newObj.(*core.Pod)
			for _, handler := range p.UpdateEventHandlers {
				handler(oldPod, newPod, nodename)
			}
			global.Logger.Infof("Update Pod %s successfully", newPod.Name)
		},
	})
	p.informer.Run(p.stopper)
}
