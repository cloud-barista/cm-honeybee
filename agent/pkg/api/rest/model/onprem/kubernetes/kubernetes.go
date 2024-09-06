package kubernetes

type Kubernetes struct {
	Nodes     []Node                 `json:"nodes"`
	Workloads map[string]interface{} `json:"workloads"`
}

type Node struct {
	Name      interface{} `json:"name,omitempty"`
	Labels    interface{} `json:"labels,omitempty"`
	Addresses interface{} `json:"addresses,omitempty"`
	NodeInfo  interface{} `json:"nodeinfo,omitempty"`
}
