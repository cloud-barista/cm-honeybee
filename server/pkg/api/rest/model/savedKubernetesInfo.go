package model

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"time"
)

type SavedKubernetesInfo struct {
	ConnectionID   string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	KubernetesData string    `gorm:"column:kubernetes_data" json:"kubernetes_data" validate:"required"`
	Status         string    `gorm:"column:status" json:"status"`
	SavedTime      time.Time `gorm:"column:saved_time" json:"saved_time"`
	// ErrorMessage says why the last import of the connection failed. It is
	// filled only in a group import response and is never stored.
	ErrorMessage string `gorm:"-" json:"error_message,omitempty"`
}

type KubernetesInfoList struct {
	Servers []kubernetes.Kubernetes `json:"servers" validate:"required"`
}
