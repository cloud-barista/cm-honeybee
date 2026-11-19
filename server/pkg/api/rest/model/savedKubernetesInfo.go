package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"time"
)

type SavedKubernetesInfo struct {
	ConnectionID   string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	KubernetesData string    `gorm:"column:kubernetes_data" json:"kubernetes_data" validate:"required"`
	Status         string    `gorm:"column:status" json:"status"`
	SavedTime      time.Time `gorm:"column:saved_time" json:"saved_time"`
	// ErrorMessage says why the last import of the connection failed. It is
	// filled only in a group import response and is never stored.
	ErrorMessage string `gorm:"-" json:"error_message,omitempty"`
}

type KubernetesInfoList struct {
	Servers []CSPKubernetesRecord `json:"servers" validate:"required"`
}

// CSPKubernetesRecord is a stored kubernetes record as GET returns it: the
// agent's kubernetes shape, which refined reads, plus, for a CSP-managed
// cluster, what the CSP reports of the cluster that the shape has no field
// for. CSP is nil in a record the agent collected and in one stored before it
// was kept.
type CSPKubernetesRecord struct {
	kubernetes.Kubernetes
	CSP *CSPKubernetesCluster `json:"csp,omitempty"`
}

// CSPKubernetesCluster is the CSP's own view of a managed cluster. The nodes of
// a managed cluster are run by the CSP and are not enumerated, so what is
// known of them is the node groups they are created from.
type CSPKubernetesCluster struct {
	SystemID    string                   `json:"system_id,omitempty"`
	Status      string                   `json:"status,omitempty"`
	CreatedTime string                   `json:"created_time,omitempty"`
	NodeGroups  []CSPKubernetesNodeGroup `json:"node_groups"`
}

// CSPKubernetesNodeGroup is one node group of a managed cluster. RootDiskSize
// is in GB, kept as text because drivers report it so.
type CSPKubernetesNodeGroup struct {
	Name            string `json:"name"`
	VMSpec          string `json:"vm_spec,omitempty"`
	Image           string `json:"image,omitempty"`
	RootDiskType    string `json:"root_disk_type,omitempty"`
	RootDiskSize    string `json:"root_disk_size,omitempty"`
	OnAutoScaling   bool   `json:"on_auto_scaling"`
	DesiredNodeSize int    `json:"desired_node_size"`
	MinNodeSize     int    `json:"min_node_size"`
	MaxNodeSize     int    `json:"max_node_size"`
	Status          string `json:"status,omitempty"`
}
