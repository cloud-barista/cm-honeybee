package kubernetes

import (
	"errors"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"k8s.io/client-go/kubernetes"

	// "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	//KubeConfigPath = os.Getenv("KUBECONFIG")
	KubeConfigPath = "/etc/kubernetes/admin.conf"
	settings       = cli.New()
)

func GetKubernetesClientSet() (*kubernetes.Clientset, error) {

	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New("get clientSet error")
	}

	return clientSet, nil
}

func GetHelmConfig(namespace string) (*action.Configuration, error) {

	//if namespace == "" {
	//	namespace = "default"
	//}

	configFlags := genericclioptions.NewConfigFlags(false)
	configFlags.KubeConfig = &KubeConfigPath
	configFlags.Namespace = &namespace

	cfg := new(action.Configuration)
	err := cfg.Init(configFlags, namespace, os.Getenv("HELM_DRIVER"), nil)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
