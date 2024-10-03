package model

const ConnectionInfoMaxLength = 200

const (
	ConnectionInfoStatusSuccess = "success"
	ConnectionInfoStatusFailed  = "failed"
)

type ConnectionInfo struct {
	ID                      string `gorm:"primaryKey" json:"id" validate:"required"`
	Name                    string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description             string `gorm:"column:description" json:"description"`
	SourceGroupID           string `gorm:"column:source_group_id" json:"source_group_id" validate:"required"`
	IPAddress               string `gorm:"column:ip_address" json:"ip_address" validate:"required"`
	SSHPort                 string `gorm:"column:ssh_port" json:"ssh_port" validate:"required"`
	User                    string `gorm:"column:user" json:"user" validate:"required"`
	Password                string `gorm:"column:password" json:"password"`
	PrivateKey              string `gorm:"column:private_key" json:"private_key"`
	PublicKey               string `gorm:"column:public_key" json:"public_key"`
	ConnectionStatus        string `gorm:"column:connection_status" json:"connection_status"`
	ConnectionFailedMessage string `gorm:"column:connection_failed_message" json:"connection_failed_message"`
	AgentStatus             string `gorm:"column:agent_status" json:"agent_status"`
	AgentFailedMessage      string `gorm:"column:agent_failed_message" json:"agent_failed_message"`
}

type CreateConnectionInfoReq struct {
	Name        string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description string `gorm:"column:description" json:"description"`
	IPAddress   string `gorm:"column:ip_address" json:"ip_address" validate:"required"`
	SSHPort     string `gorm:"column:ssh_port" json:"ssh_port" validate:"required"`
	User        string `gorm:"column:user" json:"user" validate:"required"`
	Password    string `gorm:"column:password" json:"password"`
	PrivateKey  string `gorm:"column:private_key" json:"private_key"`
}

type ListConnectionInfoRes struct {
	ConnectionInfo            []ConnectionInfo          `json:"connection_info"`
	ConnectionInfoStatusCount ConnectionInfoStatusCount `json:"connection_info_status_count"`
}
