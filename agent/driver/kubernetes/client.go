package kubernetes

import (
	"errors"
	"os"

	"k8s.io/client-go/kubernetes"

	// "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	helmclient "github.com/mittwald/go-helm-client"
)

const (
	KubeConfigPath = "/etc/kubernetes/admin.conf"
)

func KubeConfigCheck() bool {
	_, err := os.ReadFile(KubeConfigPath)
	if err != nil {
		return false
	}
	return true
}

func GetKubernetesClientSet() (*kubernetes.Clientset, error) {

	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New("get clientset error")
	}

	return clientset, nil
}

func GetHelmClientSet(ns string) (helmclient.Client, error) {

	if ns == "" {
		ns = "default"
	}

	opt := &helmclient.KubeConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        ns,
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          true,
		},
	}

	kubeConfigData, err := os.ReadFile(KubeConfigPath)
	if err != nil {
		return nil, err
	}
	opt.KubeConfig = kubeConfigData

	helmClient, err := helmclient.NewClientFromKubeConf(opt, nil)
	if err != nil {
		return nil, errors.New("get clientset error")
	}

	return helmClient, err
}
