package model

import (
	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
)

// RefinedOnpremiseInfraModel is the refined infra response. It has the JSON
// shape of cm-beetle's onpremisemodel.OnpremiseInfraModel, with each node also
// carrying its listening ports. cm-beetle reads those from
// onpremiseInfraModel.nodes[].listeningPorts, a field the imdl release this
// module depends on does not define yet, so the node type is extended here.
type RefinedOnpremiseInfraModel struct {
	OnpremiseInfraModel RefinedOnpremInfra `json:"onpremiseInfraModel" validate:"required"`
}

// RefinedOnpremInfra mirrors onpremisemodel.OnpremInfra with RefinedNodeProperty
// nodes.
type RefinedOnpremInfra struct {
	Network    inframodel.NetworkProperty     `json:"network,omitempty"`
	Nodes      []RefinedNodeProperty          `json:"nodes" validate:"required"`
	K8sCluster *inframodel.K8sClusterProperty `json:"k8sCluster,omitempty"`
	NLBs       []inframodel.NlbProperty       `json:"nlbs,omitempty"`
}

// RefinedNodeProperty is onpremisemodel.NodeProperty plus the node's listening
// ports. The embedded fields are encoded at the top level of the node object.
type RefinedNodeProperty struct {
	inframodel.NodeProperty
	ListeningPorts []ListeningPortProperty `json:"listeningPorts,omitempty"`
}

// ListeningPortProperty is one socket a node serves on: a LISTEN TCP socket or
// a bound UDP socket.
type ListeningPortProperty struct {
	Protocol string `json:"protocol" example:"tcp"` // tcp, udp
	// BindAddress is the local address the socket is bound to, e.g. "0.0.0.0",
	// "::", "127.0.0.1". It is left out when the agent that collected the node
	// predates per-socket collection and only reported the port.
	BindAddress string `json:"bindAddress,omitempty" example:"0.0.0.0"`
	Port        int    `json:"port" example:"8080"`
	ProcessName string `json:"processName,omitempty" example:"nginx"`
}
