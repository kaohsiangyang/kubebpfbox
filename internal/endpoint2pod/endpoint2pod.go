package endpoint2pod

import (
	"kubebpfbox/global"
	"kubebpfbox/internal/k8s"
	"sync"

	core "k8s.io/api/core/v1"
)

type Endpoint struct {
	IP   string
	Port int32
}

type Provider struct {
	PodName     string
	NodeName    string
	NameSpace   string
	ServiceName string
}

type Endpoint2Pod struct {
	serviceController   *k8s.ServiceController
	Providers           map[Endpoint]Provider
	AddEventHandlers    []func(newEndpoints *Endpoint, newProvider *Provider) error
	DeleteEventHandlers []func(oldEndpoints *Endpoint, oldProvider *Provider) error
}

var endpoint2Pod *Endpoint2Pod
var once sync.Once

// GetEndpoint2Pod returns a singleton Endpoint2Pod
func GetEndpoint2Pod() *Endpoint2Pod {
	once.Do(func() {
		endpoint2Pod = &Endpoint2Pod{
			serviceController: k8s.GetServiceController(),
			Providers:         make(map[Endpoint]Provider),
		}
	})
	return endpoint2Pod
}

// GetPodByEndpoint returns a pod by endpoint
func (i *Endpoint2Pod) GetPodByEndpoint(ip string, port int32) (provider *Provider, ok bool) {
	if provider, ok := i.Providers[Endpoint{ip, port}]; ok {
		return &provider, true
	}
	return nil, false
}

// addEndpoints adds endpoints to endpoint2Pod
func (i *Endpoint2Pod) addEndpoints(newEndpoints *core.Endpoints, nodeName string) {
	for _, subsets := range newEndpoints.Subsets {
		for _, address := range subsets.Addresses {
			for _, port := range subsets.Ports {
				if address.NodeName != nil && *address.NodeName == nodeName {
					provider := Provider{
						PodName:     address.TargetRef.Name,
						NodeName:    nodeName,
						NameSpace:   address.TargetRef.Namespace,
						ServiceName: newEndpoints.Name,
					}

					GetEndpoint2Pod().Providers[Endpoint{address.IP, port.Port}] = provider
					for _, handler := range GetEndpoint2Pod().AddEventHandlers {
						handler(&Endpoint{address.IP, port.Port}, &provider)
					}
					global.Logger.Infof("Add Endpoint: [%s:%d] in Service %s successfully",
						address.IP, port.Port, newEndpoints.Name)
				}
			}
		}
	}
}

// deleteEndpoints deletes endpoints from endpoint2Pod
func (i *Endpoint2Pod) deleteEndpoints(oldEndpoints *core.Endpoints, nodeName string) {
	for _, subsets := range oldEndpoints.Subsets {
		for _, address := range subsets.Addresses {
			for _, port := range subsets.Ports {
				if address.NodeName != nil && *address.NodeName == nodeName {
					delete(GetEndpoint2Pod().Providers, Endpoint{address.IP, port.Port})
					for _, handler := range GetEndpoint2Pod().DeleteEventHandlers {
						handler(&Endpoint{address.IP, port.Port}, &Provider{
							PodName:     address.TargetRef.Name,
							NodeName:    nodeName,
							NameSpace:   address.TargetRef.Namespace,
							ServiceName: oldEndpoints.Name,
						})
					}
					global.Logger.Infof("Delete Endpoint: [%s:%d] in Service %s successfully",
						address.IP, port.Port, oldEndpoints.Name)
				}
			}
		}
	}
}

// updateEndpoints updates endpoints in endpoint2Pod
func (i *Endpoint2Pod) updateEndpoints(oldEndpoints *core.Endpoints, newEndpoints *core.Endpoints, nodeName string) {
	for _, subsets := range oldEndpoints.Subsets {
		for _, address := range subsets.Addresses {
			for _, port := range subsets.Ports {
				if address.NodeName != nil && *address.NodeName == nodeName {
					delete(GetEndpoint2Pod().Providers, Endpoint{address.IP, port.Port})
					for _, handler := range GetEndpoint2Pod().DeleteEventHandlers {
						handler(&Endpoint{address.IP, port.Port}, &Provider{
							PodName:     address.TargetRef.Name,
							NodeName:    nodeName,
							NameSpace:   address.TargetRef.Namespace,
							ServiceName: oldEndpoints.Name,
						})
					}
					global.Logger.Infof("Delete Endpoint: [%s:%d] in Service %s successfully",
						address.IP, port.Port, oldEndpoints.Name)
				}
			}
		}
	}
	for _, subsets := range newEndpoints.Subsets {
		for _, address := range subsets.Addresses {
			for _, port := range subsets.Ports {
				if address.NodeName != nil && *address.NodeName == nodeName {
					provider := Provider{
						PodName:     address.TargetRef.Name,
						NodeName:    nodeName,
						NameSpace:   address.TargetRef.Namespace,
						ServiceName: newEndpoints.Name,
					}
					GetEndpoint2Pod().Providers[Endpoint{address.IP, port.Port}] = provider
					for _, handler := range GetEndpoint2Pod().AddEventHandlers {
						handler(&Endpoint{address.IP, port.Port}, &provider)
					}
					global.Logger.Infof("Add Endpoint: [%s:%d] in Service %s successfully",
						address.IP, port.Port, newEndpoints.Name)
				}
			}
		}
	}
}

// Registry registers endpoint2Pod to service controller
func (i *Endpoint2Pod) Registry() {
	i.serviceController.AddEventHandlers = append(i.serviceController.AddEventHandlers, i.addEndpoints)
	i.serviceController.DeleteEventHandlers = append(i.serviceController.DeleteEventHandlers, i.deleteEndpoints)
	i.serviceController.UpdateEventHandlers = append(i.serviceController.UpdateEventHandlers, i.updateEndpoints)
	global.Logger.Info("Registry endpoint2Pod to service controller successfully")
}
