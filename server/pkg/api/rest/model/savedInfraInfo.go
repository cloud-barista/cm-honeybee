package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"time"
)

type SavedInfraInfo struct {
	ConnectionID string `gorm:"primaryKey" json:"connection_id" validate:"required"`
	InfraData    string `gorm:"column:infra_data" json:"infra_data"`
	// CSPData holds the CSP-side VM information (cb-spider) as JSON, kept separate
	// from InfraData (agent-collected) so importing the agent data does not
	// overwrite the CSP data and vice versa.
	CSPData   string    `gorm:"column:csp_data" json:"csp_data,omitempty"`
	Status    string    `gorm:"column:status" json:"status"`
	SavedTime time.Time `gorm:"column:saved_time" json:"saved_time"`
	// ErrorMessage says why the last import of the connection failed. It is
	// filled only in a group import response and is never stored.
	ErrorMessage string `gorm:"-" json:"error_message,omitempty"`
}

type InfraInfoList struct {
	Servers []infra.Infra `json:"servers" validate:"required"`
}
