package kubernetes

import (
	"time"
)

type Kubernetes struct {
	NodeCount NodeCount              `json:"node_count"`
	Nodes     []Node                 `json:"nodes"`
	Workloads map[string]interface{} `json:"workloads"`
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
	CPU              int `json:"cpu"`               // cores
	Memory           int `json:"memory"`            // MiB
	EphemeralStorage int `json:"ephemeral_storage"` // MiB
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
