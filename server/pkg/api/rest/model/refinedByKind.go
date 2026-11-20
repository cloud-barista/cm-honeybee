package model

import (
	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
)

// RefinedK8sClusterModel is the refined Kubernetes cluster of a connection or
// of a source group. K8sCluster is what infra/refined gives as k8sCluster.
type RefinedK8sClusterModel struct {
	// ConnectionID is the connection the cluster was collected from, empty
	// when there is no cluster.
	ConnectionID string `json:"connection_id,omitempty"`
	// Managed is true for a cluster the CSP runs, collected from a csp k8s
	// connection, and false for one the agent collected through a kubeconfig.
	Managed    bool                           `json:"managed"`
	K8sCluster *inframodel.K8sClusterProperty `json:"k8sCluster"`
	// CSP is the CSP's own view of a managed cluster, its node groups among
	// it. It is left out for a cluster the agent collected and for a record
	// stored before it was kept.
	CSP *CSPKubernetesCluster `json:"csp,omitempty"`
}

// RefinedNLBModel is the refined load balancers of a connection or of a
// source group: the NLBs found in HAProxy on the hosts and the CSP NLBs, the
// same list infra/refined gives as nlbs.
type RefinedNLBModel struct {
	NLBs []inframodel.NlbProperty `json:"nlbs"`
}
