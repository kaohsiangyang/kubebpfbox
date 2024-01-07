package k8s

import (
	"kubebpfbox/global"
	"os"
	"sync"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type ServiceController struct {
	stopper             chan struct{}
	informer            cache.SharedIndexInformer
	AddEventHandlers    []func(newEndpoints *core.Endpoints, nodeName string)
	DeleteEventHandlers []func(oldEndpoints *core.Endpoints, nodeName string)
	UpdateEventHandlers []func(oldEndpoints *core.Endpoints, newEndpoints *core.Endpoints, nodeName string)
}

var serviceController *ServiceController
var onceService sync.Once

func GetServiceController() *ServiceController {
	onceService.Do(func() {
		factory := NewInformerFactory().Factory

		informer := factory.Core().V1().Endpoints().Informer()

		serviceController = &ServiceController{
			stopper:  make(chan struct{}),
			informer: informer,
		}
	})
	return serviceController
}

// Run starts the service controller
func (p *ServiceController) Run() {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		global.Logger.Fatalf("Environment variables 'NODE_NAME' need to set !")
	}

	p.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEndpoints := obj.(*core.Endpoints)
			for _, handler := range p.AddEventHandlers {
				handler(newEndpoints, nodeName)
			}
			global.Logger.Infof("Add Service %s successfully", newEndpoints.Name)
		},
		DeleteFunc: func(obj interface{}) {
			oldEndpoints := obj.(*core.Endpoints)
			for _, handler := range p.DeleteEventHandlers {
				handler(oldEndpoints, nodeName)
			}
			global.Logger.Infof("Delete Service %s successfully", oldEndpoints.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldEndpoints := oldObj.(*core.Endpoints)
			newEndpoints := newObj.(*core.Endpoints)
			for _, handler := range p.UpdateEventHandlers {
				handler(oldEndpoints, newEndpoints, nodeName)
			}
			global.Logger.Infof("Update Service %s successfully", newEndpoints.Name)
		},
	})
	p.informer.Run(p.stopper)
}
