package k8s

import (
	"log"
	"os"
	"path/filepath"

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

// NewInformerFactory return a informer factory
func NewInformerFactory() *InformerFactory {
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
		log.Fatal(err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// create the factory
	factory := informers.NewSharedInformerFactory(clientset, 0)

	return &InformerFactory{
		Config:    config,
		Clientset: clientset,
		Factory:   factory,
	}
}

// OutClusterConfig return a config for out of cluster
func outClusterConfig() (config *rest.Config, err error) {
	var kubeconfig string
	if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		// TODO: 通过config文件读取配置文件的地址
		kubeconfig = "/root/.kube/config"
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
