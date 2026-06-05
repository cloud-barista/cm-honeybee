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

// getKubeConfigPath prefers the KUBECONFIG environment variable and falls back
// to the kubeadm admin config available on control plane nodes.
func getKubeConfigPath() string {
	if kubeConfigPath := os.Getenv("KUBECONFIG"); kubeConfigPath != "" {
		return kubeConfigPath
	}
	return "/etc/kubernetes/admin.conf"
}

var (
	KubeConfigPath = getKubeConfigPath()
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
