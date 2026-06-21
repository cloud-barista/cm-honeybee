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

// KeyValue mirrors cb-spider's KeyValue and is also used for CSP credential KV
// stored on a SourceGroup. Values are stored RSA-encrypted (base64) for csp-type
// SourceGroups.
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// KeyValueList is a JSON-serialized GORM column type.
type KeyValueList []KeyValue

func (k KeyValueList) Value() (driver.Value, error) {
	return json.Marshal(k)
}

func (k *KeyValueList) Scan(value interface{}) error {
	if value == nil {
		*k = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for KeyValueList")
	}
	return json.Unmarshal(bytes, k)
}

type SourceGroup struct {
	ID          string     `gorm:"primaryKey" json:"id" validate:"required"`
	Name        string     `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" validate:"required"`
	Description string     `gorm:"column:description" json:"description"`
	TargetInfo  TargetInfo `gorm:"column:target_info" json:"target_info"`

	// Type discriminates how this group's connections are collected.
	// Allowed: "ssh" (default, on-prem) or "csp" (cb-spider backed).
	Type string `gorm:"column:type;default:ssh" json:"type"`

	// CSP fields — populated only when Type == "csp".
	// Credential is stored RSA-encrypted at rest. It is decrypted on demand and
	// registered to cb-spider only transiently (per discovery/collection call);
	// honeybee is the single source of truth, so no spider connection name is kept.
	ProviderName string       `gorm:"column:provider_name" json:"provider_name,omitempty"`
	RegionName   string       `gorm:"column:region_name" json:"region_name,omitempty"`
	Credential   KeyValueList `gorm:"column:credential" json:"credential,omitempty"`
}

type CreateSourceGroupReq struct {
	Name           string                    `json:"name" validate:"required"`
	Description    string                    `json:"description"`
	ConnectionInfo []CreateConnectionInfoReq `json:"connection_info"`

	// CSP fields — required when Type == "csp".
	Type         string     `json:"type"`
	ProviderName string     `json:"provider_name,omitempty"`
	RegionName   string     `json:"region_name,omitempty"`
	Credential   []KeyValue `json:"credential,omitempty"`
}

type UpdateSourceGroupReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`

	// CSP fields — only honored for CSP groups.
	RegionName string     `json:"region_name,omitempty"`
	Credential []KeyValue `json:"credential,omitempty"`
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
	Type                      string                    `json:"type"`
	ProviderName              string                    `json:"provider_name,omitempty"`
	RegionName                string                    `json:"region_name,omitempty"`
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
