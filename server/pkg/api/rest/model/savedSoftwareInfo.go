package model

import (
	"time"
)

type SavedSoftwareInfo struct {
	ConnectionUUID string    `gorm:"primaryKey" json:"connection_uuid" validate:"required"`
	SoftwareData   string    `gorm:"column:software_data" json:"software_data" validate:"required"`
	Status         string    `gorm:"column:status" json:"status"`
	SavedTime      time.Time `gorm:"column:saved_time" json:"saved_time"`
}
