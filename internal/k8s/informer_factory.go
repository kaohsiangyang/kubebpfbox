package k8s

import (
	"kubebpfbox/global"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type InformerFactory struct {
	Config    *rest.Config
	Clientset *kubernetes.Clientset
	Factory   informers.SharedInformerFactory
}

var informerFactory *InformerFactory
var onceInformer sync.Once

// GetInformerFactory return a informer factory
func GetInformerFactory() *InformerFactory {
	onceInformer.Do(func() {
		// get the kube config
		var config *rest.Config
		var err error
		incluster := os.Getenv("IN_CLUSTER")
		if incluster == "true" {
			config, err = rest.InClusterConfig()
		} else {
			config, err = outClusterConfig()
		}
		if err != nil {
			global.Logger.Fatal("Parse cluster config failed: ", err)
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			global.Logger.Fatal("Create clientset failed: ", err)
		}

		// create the factory
		factory := informers.NewSharedInformerFactory(clientset, 0)

		informerFactory = &InformerFactory{
			Config:    config,
			Clientset: clientset,
			Factory:   factory,
		}
	})
	return informerFactory
}

// OutClusterConfig return a config for out of cluster
func outClusterConfig() (config *rest.Config, err error) {
	var kubeconfig string
	if global.ClusterSetting.ConfigPath != "" {
		kubeconfig = global.ClusterSetting.ConfigPath
	} else if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	return
}

// HomeDir return the home dir
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
