package kubernetes

import (
	"context"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"

	"github.com/jollaman999/utils/logger"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

// kubeadmClusterConfiguration is a partial view of the kubeadm
// ClusterConfiguration stored in the kube-system/kubeadm-config ConfigMap.
type kubeadmClusterConfiguration struct {
	ClusterName       string `yaml:"clusterName"`
	KubernetesVersion string `yaml:"kubernetesVersion"`
	Networking        struct {
		PodSubnet     string `yaml:"podSubnet"`
		ServiceSubnet string `yaml:"serviceSubnet"`
	} `yaml:"networking"`
}

// parseKubeadmConfig fills cluster metadata from the kubeadm-config ConfigMap.
// Non-kubeadm clusters do not have it, so a lookup failure is not an error.
func parseKubeadmConfig(clientset *k8sclient.Clientset, cluster *kubernetes.Cluster) {
	configMap, err := clientset.CoreV1().ConfigMaps("kube-system").
		Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		logger.Println(logger.DEBUG, false, "Kubernetes: kubeadm-config ConfigMap not available: "+err.Error())
		return
	}

	data, ok := configMap.Data["ClusterConfiguration"]
	if !ok {
		return
	}

	var config kubeadmClusterConfiguration
	if err := yaml.Unmarshal([]byte(data), &config); err != nil {
		logger.Println(logger.WARN, true, "Kubernetes: failed to parse kubeadm ClusterConfiguration: "+err.Error())
		return
	}

	cluster.Name = config.ClusterName
	cluster.PodCIDR = config.Networking.PodSubnet
	cluster.ServiceCIDR = config.Networking.ServiceSubnet
	if cluster.Version == "" {
		cluster.Version = strings.TrimPrefix(config.KubernetesVersion, "v")
	}
}

// findPodsFlag scans the command line of the given pods' containers for
// "--<flag>=<value>" and returns the first value found.
func findPodsFlag(pods []corev1.Pod, flag string) string {
	prefix := flag + "="

	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			cmdline := append(append([]string{}, container.Command...), container.Args...)
			for _, arg := range cmdline {
				if value, ok := strings.CutPrefix(arg, prefix); ok {
					return value
				}
			}
		}
	}

	return ""
}

func listControlPlanePods(clientset *k8sclient.Clientset, component string) []corev1.Pod {
	pods, err := clientset.CoreV1().Pods("kube-system").
		List(context.TODO(), metav1.ListOptions{LabelSelector: "component=" + component})
	if err != nil {
		logger.Println(logger.DEBUG, false, "Kubernetes: failed to list "+component+" pods: "+err.Error())
		return nil
	}

	return pods.Items
}

// parseControlPlaneFlags fills network related cluster metadata from the
// command line flags of the static control plane pods. Used as the primary
// source for the NodePort range and as a fallback for the CIDRs when the
// kubeadm-config ConfigMap is not available.
func parseControlPlaneFlags(clientset *k8sclient.Clientset, cluster *kubernetes.Cluster) {
	apiServerPods := listControlPlanePods(clientset, "kube-apiserver")

	if value := findPodsFlag(apiServerPods, "--service-node-port-range"); value != "" {
		cluster.NodePortRange = value
	}
	if cluster.ServiceCIDR == "" {
		cluster.ServiceCIDR = findPodsFlag(apiServerPods, "--service-cluster-ip-range")
	}

	if cluster.PodCIDR == "" {
		controllerManagerPods := listControlPlanePods(clientset, "kube-controller-manager")
		cluster.PodCIDR = findPodsFlag(controllerManagerPods, "--cluster-cidr")
	}
}

// cniPatterns maps DaemonSet name substrings to CNI plugin names.
// Order matters: canal ships calico and flannel components, so it is checked first.
var cniPatterns = []struct {
	substring string
	plugin    string
}{
	{"canal", "canal"},
	{"calico", "calico"},
	{"cilium", "cilium"},
	{"flannel", "flannel"},
	{"weave", "weave"},
	{"kube-router", "kube-router"},
	{"antrea", "antrea"},
}

// detectCNIPlugin guesses the CNI plugin from DaemonSet names across all
// namespaces (e.g., calico-node in kube-system, kube-flannel-ds in kube-flannel).
func detectCNIPlugin(clientset *k8sclient.Clientset, cluster *kubernetes.Cluster) {
	daemonSets, err := clientset.AppsV1().DaemonSets("").
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Println(logger.WARN, true, "Kubernetes: failed to list DaemonSets: "+err.Error())
		return
	}

	for _, pattern := range cniPatterns {
		for _, daemonSet := range daemonSets.Items {
			if strings.Contains(strings.ToLower(daemonSet.Name), pattern.substring) {
				cluster.CNIPlugin = pattern.plugin
				return
			}
		}
	}
}

// GetClusterInfo collects cluster-wide metadata (name, version, network CIDRs,
// CNI plugin and NodePort range) needed by the refined on-premise model.
func GetClusterInfo() (kubernetes.Cluster, error) {
	var cluster kubernetes.Cluster

	clientset, err := GetKubernetesClientSet()
	if err != nil {
		logger.Println(logger.ERROR, true, "Kubernetes Connection Error: "+err.Error())
		return cluster, err
	}

	if versionInfo, err := clientset.Discovery().ServerVersion(); err == nil {
		cluster.Version = strings.TrimPrefix(versionInfo.GitVersion, "v")
	} else {
		logger.Println(logger.WARN, true, "Kubernetes: failed to get server version: "+err.Error())
	}

	parseKubeadmConfig(clientset, &cluster)
	parseControlPlaneFlags(clientset, &cluster)
	detectCNIPlugin(clientset, &cluster)

	if cluster.Name == "" {
		cluster.Name = "kubernetes" // kubeadm default cluster name
	}
	if cluster.NodePortRange == "" {
		cluster.NodePortRange = "30000-32767" // Kubernetes default service node port range
	}

	return cluster, nil
}
