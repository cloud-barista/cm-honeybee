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

	// Type discriminates how this group's connections are collected. Allowed:
	// "onprem" or "csp" (cb-spider backed) for infrastructure, and "fs", "db" or
	// "minio" for data migration sources; see common.IsOnpremType. "ssh" and
	// the empty value are the earlier spelling of "onprem" and are still
	// accepted. The stored default stays "ssh" rather than "onprem" because it
	// is what clients already compare against; the request handler writes it
	// explicitly when the caller omits the field.
	Type string `gorm:"column:type;default:ssh" json:"type"`

	// ProviderName is where the sources are hosted. csp: the cb-spider provider.
	// minio: picks the S3 endpoint form. db: recorded only, nothing is derived
	// from it. Both minio and db take the providers in provider.go, "onprem" for
	// the operator's own servers.
	// RegionName: csp and minio only. minio requires it for providers whose
	// endpoint or signing carries the region (see regionRequired).
	// Credential is csp only. It lives in OpenBao, never in this table: the
	// column is cleared on write and rehydrated on demand. It is handed to the
	// CSP driver per discovery/collection call and registered nowhere else;
	// honeybee is the single source of truth, so no connection name is kept.
	ProviderName string       `gorm:"column:provider_name" json:"provider_name,omitempty"`
	RegionName   string       `gorm:"column:region_name" json:"region_name,omitempty"`
	Credential   KeyValueList `gorm:"column:credential" json:"credential,omitempty"`
}

type CreateSourceGroupReq struct {
	Name           string                    `json:"name" validate:"required"`
	Description    string                    `json:"description"`
	ConnectionInfo []CreateConnectionInfoReq `json:"connection_info"`

	// Type: "onprem" (default; legacy "ssh") | "csp" | "fs" | "db" | "minio".
	// provider_name is required for csp, minio and db; region_name for csp and
	// for minio providers that need it; credential for csp only.
	Type         string     `json:"type"`
	ProviderName string     `json:"provider_name,omitempty"`
	RegionName   string     `json:"region_name,omitempty"`
	Credential   []KeyValue `json:"credential,omitempty"`
}

type UpdateSourceGroupReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`

	// RegionName is honored for csp and minio groups, Credential for csp only.
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
