package model

type ConnectionInfo struct {
	ID            string `gorm:"primaryKey" json:"id" validate:"required"`
	Name          string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description   string `gorm:"column:description" json:"description"`
	SourceGroupID string `gorm:"column:source_group_id" json:"source_group_id" validate:"required"`
	IPAddress     string `gorm:"column:ip_address" json:"ip_address" validate:"required"`
	SSHPort       string `gorm:"column:ssh_port" json:"ssh_port" validate:"required"`
	User          string `gorm:"column:user" json:"user" validate:"required"`
	Password      string `gorm:"column:password" json:"password"`
	PrivateKey    string `gorm:"column:private_key" json:"private_key"`
	PublicKey     string `gorm:"column:public_key" json:"public_key"`
	Status        string `gorm:"column:status" json:"status"`
	FailedMessage string `gorm:"column:failed_message" json:"failed_message"`
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
