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

type SourceGroup struct {
	ID          string     `gorm:"primaryKey" json:"id" validate:"required"`
	Name        string     `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description string     `gorm:"column:description" json:"description"`
	TargetInfo  TargetInfo `gorm:"column:target_info" json:"target_info"`
}

type CreateSourceGroupReq struct {
	Name        string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description string `gorm:"column:description" json:"description"`
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
		return errors.New("Invalid type for TargetInfo")
	}
	return json.Unmarshal(bytes, t)
}

type RegisterTargetInfoReq struct {
	ResourceType  string `json:"resourceType" validate:"required"`
	ID            string `json:"id" validate:"required"`
	UID           string `json:"uid"`
	Name          string `json:"name"`
	TargetStatus  string `json:"targetStatus"`
	TargetAction  string `json:"targetAction"`
	Label         string `json:"label"`
	SystemLabel   string `json:"systemLabel"`
	SystemMessage string `json:"systemMessage"`
	Description   string `json:"description"`
}
