package model

import (
	"time"
)

type SavedBenchmarkInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id"`
	Benchmark    string    `gorm:"column:benchmark;type:longtext" json:"benchmark,omitempty"`
	Status       string    `json:"status"`
	SavedTime    time.Time `json:"saved_time"`
}

type Benchmark struct {
	Type string        `json:"type"`
	Data BenchmarkData `json:"data" gorm:"embedded"`
}

type BenchmarkData struct {
	Desc    string `json:"desc"`
	Elapsed string `json:"elapsed"`
	Result  string `json:"result"`
	SpecID  string `json:"specid"`
	Unit    string `json:"unit"`
}
