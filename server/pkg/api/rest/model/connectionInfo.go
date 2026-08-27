package model

const ConnectionInfoMaxLength = 200

const (
	ConnectionInfoStatusSuccess = "success"
	ConnectionInfoStatusFailed  = "failed"
)

type ConnectionInfo struct {
	ID            string `gorm:"primaryKey" json:"id" validate:"required"`
	Name          string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description   string `gorm:"column:description" json:"description"`
	SourceGroupID string `gorm:"column:source_group_id" json:"source_group_id" validate:"required"`

	// SSH fields — required when parent SourceGroup.Type == "ssh".
	IPAddress  string `gorm:"column:ip_address" json:"ip_address,omitempty"`
	SSHPort    string `gorm:"column:ssh_port" json:"ssh_port,omitempty"`
	User       string `gorm:"column:user" json:"user,omitempty"`
	Password   string `gorm:"column:password" json:"password,omitempty"`
	PrivateKey string `gorm:"column:private_key" json:"private_key,omitempty"`
	PublicKey  string `gorm:"column:public_key" json:"public_key,omitempty"`

	// Kubeconfig — on-prem k8s credential (Type onprem + ResourceType k8s).
	Kubeconfig string `gorm:"column:kubeconfig" json:"kubeconfig,omitempty"`

	// CSP fields — required when parent SourceGroup.Type == "csp".
	// ResourceType: "vm" | "k8s" | "object_storage".
	ResourceType string `gorm:"column:resource_type" json:"resource_type,omitempty"`
	ResourceID   string `gorm:"column:resource_id" json:"resource_id,omitempty"`
	// Zone overrides the CSP zone for this connection's cb-spider lookups.
	// Empty → provider default (see buildRegionKV).
	Zone string `gorm:"column:zone" json:"zone,omitempty"`

	ConnectionStatus        string `gorm:"column:connection_status" json:"connection_status"`
	ConnectionFailedMessage string `gorm:"column:connection_failed_message" json:"connection_failed_message"`
	AgentStatus             string `gorm:"column:agent_status" json:"agent_status"`
	AgentFailedMessage      string `gorm:"column:agent_failed_message" json:"agent_failed_message"`
}

type CreateConnectionInfoReq struct {
	Name        string `json:"name" mapstructure:"name" validate:"required"`
	Description string `json:"description"`

	// SSH fields — required when parent SourceGroup.Type == "ssh".
	IPAddress  string `json:"ip_address,omitempty"`
	SSHPort    string `json:"ssh_port,omitempty"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`

	// Kubeconfig — required when on-prem + ResourceType "k8s".
	Kubeconfig string `json:"kubeconfig,omitempty"`

	// CSP fields — required when parent SourceGroup.Type == "csp".
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	// Zone (optional) overrides the CSP zone for this connection. Empty → default.
	Zone string `json:"zone,omitempty"`
}

type ListConnectionInfoRes struct {
	ConnectionInfo            []ConnectionInfo          `json:"connection_info"`
	ConnectionInfoStatusCount ConnectionInfoStatusCount `json:"connection_info_status_count"`
}
