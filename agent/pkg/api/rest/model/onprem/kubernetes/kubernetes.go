package kubernetes

import (
	"time"
)

type Kubernetes struct {
	Cluster   Cluster                `json:"cluster"`
	NodeCount NodeCount              `json:"node_count"`
	Nodes     []Node                 `json:"nodes"`
	Workloads map[string]interface{} `json:"workloads"`
}

// Cluster holds cluster-wide metadata used by the refined on-premise model.
type Cluster struct {
	Name          string `json:"name,omitempty"`            // Cluster name (e.g., kubeadm ClusterConfiguration clusterName)
	Version       string `json:"version,omitempty"`         // Kubernetes version (e.g., "1.29.3")
	PodCIDR       string `json:"pod_cidr,omitempty"`        // Pod network CIDR (e.g., "10.244.0.0/16")
	ServiceCIDR   string `json:"service_cidr,omitempty"`    // Service network CIDR (e.g., "10.96.0.0/12")
	CNIPlugin     string `json:"cni_plugin,omitempty"`      // CNI plugin name (e.g., "calico", "flannel", "cilium")
	NodePortRange string `json:"node_port_range,omitempty"` // NodePort range (e.g., "30000-32767")
}

type NodeCount struct {
	Total        int `json:"total"`
	ControlPlane int `json:"control_plane"`
	Worker       int `json:"worker"`
}

type NodeType string

const (
	NodeTypeControlPlane NodeType = "control-plane"
	NodeTypeWorker       NodeType = "worker"
)

type Node struct {
	Type      NodeType    `json:"type"`
	Name      interface{} `json:"name,omitempty"`
	Labels    interface{} `json:"labels,omitempty"`
	Addresses interface{} `json:"addresses,omitempty"`
	NodeSpec  NodeSpec    `json:"node_spec,omitempty"`
	NodeInfo  interface{} `json:"node_info,omitempty"`
}

type NodeSpec struct {
	CPU              int       `json:"cpu"`               // cores
	Memory           int       `json:"memory"`            // MiB
	EphemeralStorage int       `json:"ephemeral_storage"` // MiB
	GPU              []NodeGPU `json:"gpu,omitempty"`
}

// NodeGPU is one GPU extended resource a node advertises, such as
// nvidia.com/gpu or amd.com/gpu. Kubernetes never reports GPUs as a core
// resource, so this is the only place a cluster states that it has any.
//
// Capacity and Allocatable are device counts, not cores. The remaining fields
// come from the labels a device plugin's feature discovery puts on the node,
// so they are empty on a cluster that runs the plain device plugin without it.
type NodeGPU struct {
	ResourceName  string `json:"resource_name"`
	Capacity      int64  `json:"capacity"`
	Allocatable   int64  `json:"allocatable"`
	Vendor        string `json:"vendor,omitempty"`
	Product       string `json:"product,omitempty"`
	DriverVersion string `json:"driver_version,omitempty"`
	Memory        int    `json:"memory,omitempty"` // MiB, per device
	MIGCapable    string `json:"mig_capable,omitempty"`
	MIGStrategy   string `json:"mig_strategy,omitempty"`
}

type Helm struct {
	Repo    []Repo    `json:"repo"`
	Release []Release `json:"release"`
}

type Repo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Release struct {
	Name             string    `json:"name"`
	Namespace        string    `json:"namespace"`
	Revision         int       `json:"revision"`
	Updated          time.Time `json:"updated"`
	Status           string    `json:"status"`
	ChartNameVersion string    `json:"chart"`
	AppVersion       string    `json:"app_version"`
}
