package onprem

type ConnectionInfo struct {
	UUID          string `gorm:"primaryKey" json:"uuid" validate:"required"`
	GroupUUID     string `gorm:"column:group_uuid" json:"group_uuid" validate:"required"`
	IPAddress     string `gorm:"column:ip_address" json:"ip_address" validate:"required"`
	SSHPort       int    `gorm:"column:ssh_port" json:"ssh_port" validate:"required"`
	User          string `gorm:"column:user" json:"user" validate:"required"`
	Password      string `gorm:"column:password" json:"password"`
	PrivateKey    string `gorm:"column:private_key" json:"private_key"`
	PublicKey     string `gorm:"column:public_key" json:"public_key"`
	Type          string `gorm:"column:type" json:"type"`
	Status        string `gorm:"column:status" json:"status"`
	FailedMessage string `gorm:"column:failed_message" json:"failed_message"`
}
