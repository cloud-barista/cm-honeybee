package model

import (
	"time"
)

type SavedInfraInfo struct {
	ConnectionUUID string    `gorm:"primaryKey" json:"connection_uuid" validate:"required"`
	InfraData      string    `gorm:"column:infra_data" json:"infra_data" validate:"required"`
	Status         string    `gorm:"column:status" json:"status"`
	SavedTime      time.Time `gorm:"column:saved_time" json:"saved_time"`
}
