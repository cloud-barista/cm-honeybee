package infra

import (
	"errors"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClientSet() (*kubernetes.Clientset, error) {

	configPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, errors.New(".kube/config: no such file or directory")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New("get clientset error")
	}

	return clientset, nil
}
