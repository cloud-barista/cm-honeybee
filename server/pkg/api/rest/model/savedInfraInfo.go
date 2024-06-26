package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"time"
)

type SavedInfraInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	InfraData    string    `gorm:"column:infra_data" json:"infra_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
}

type InfraInfoList struct {
	Servers []infra.Infra `json:"servers" validate:"required"`
}
