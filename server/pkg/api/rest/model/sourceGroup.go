package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type TargetInfo struct {
	NSID  string `json:"ns_id"`
	MCIID string `json:"mci_id"`
}

type RegisterTargetInfoReq struct {
	ResourceType string `json:"resourceType" validate:"required"`
	ID           string `json:"id" validate:"required"`
	Label        struct {
		SysNamespace string `json:"sys.namespace" validate:"required"`
	} `json:"label" validate:"required"`
}

type SourceGroup struct {
	ID          string     `gorm:"primaryKey" json:"id" validate:"required"`
	Name        string     `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" validate:"required"`
	Description string     `gorm:"column:description" json:"description"`
	TargetInfo  TargetInfo `gorm:"column:target_info" json:"target_info"`
}

type CreateSourceGroupReq struct {
	Name           string                    `json:"name" validate:"required"`
	Description    string                    `json:"description"`
	ConnectionInfo []CreateConnectionInfoReq `json:"connection_info"`
}

type UpdateSourceGroupReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type ConnectionInfoStatusCount struct {
	CountConnectionSuccess int `json:"count_connection_success"`
	CountConnectionFailed  int `json:"count_connection_failed"`
	CountAgentSuccess      int `json:"count_agent_success"`
	CountAgentFailed       int `json:"count_agent_failed"`
	ConnectionInfoTotal    int `json:"connection_info_total"`
}

type SourceGroupRes struct {
	ID                        string                    `json:"id" validate:"required"`
	Name                      string                    `json:"name" validate:"required"`
	Description               string                    `json:"description"`
	ConnectionInfoStatusCount ConnectionInfoStatusCount `json:"connection_info_status_count"`
}

type ListSourceGroupRes struct {
	SourceGroup               []SourceGroupRes          `json:"source_group"`
	ConnectionInfoStatusCount ConnectionInfoStatusCount `json:"connection_info_status_count"`
}

func (t TargetInfo) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TargetInfo) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for TargetInfo")
	}
	return json.Unmarshal(bytes, t)
}

func (c ConnectionInfoStatusCount) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *ConnectionInfoStatusCount) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for ConnectionInfoStatusCount")
	}
	return json.Unmarshal(bytes, c)
}
