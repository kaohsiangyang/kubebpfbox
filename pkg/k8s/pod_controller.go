package k8s

import (
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

func NewPodController() *PodController {
	factory := NewInformerFactory().Factory
	informer := factory.Core().V1().Pods().Informer()
	return &PodController{
		stopper:             make(chan struct{}),
		informer:            informer,
		AddEventHandlers:    make([]func(newPod *core.Pod, nodename string), 2),
		DeleteEventHandlers: make([]func(oldPod *core.Pod, nodename string), 2),
		UpdateEventHandlers: make([]func(oldPod *core.Pod, newPod *core.Pod, nodename string), 2),
	}
}

func (p *PodController) Run() {
	p.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*core.Pod)
			for _, handler := range p.AddEventHandlers {
				handler(pod, pod.Spec.NodeName)
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*core.Pod)
			for _, handler := range p.DeleteEventHandlers {
				handler(pod, pod.Spec.NodeName)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*core.Pod)
			newPod := newObj.(*core.Pod)
			for _, handler := range p.UpdateEventHandlers {
				handler(oldPod, newPod, newPod.Spec.NodeName)
			}
		},
	})
	p.informer.Run(p.stopper)
}
