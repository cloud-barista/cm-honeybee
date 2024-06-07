package model

import "time"

type SavedBenchmarkInfo struct {
	ConnectionID  string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	BenchmarkData string    `gorm:"column:benchmark_data" json:"benchmark_data" validate:"required"`
	Status        string    `gorm:"column:status" json:"status"`
	SavedTime     time.Time `gorm:"column:saved_time" json:"saved_time"`
}

type Benchmark struct {
	Type string        `json:"type"`
	Data BenchmarkData `json:"data"`
}

type BenchmarkData struct {
	Desc    string `json:"desc"`
	Elapsed string `json:"elapsed"`
	Result  string `json:"result"`
	SpecID  string `json:"specid"`
	Unit    string `json:"unit"`
}

// type BenchmarkInfo struct {
// 	ConnectionUUID string `gorm:"primaryKey" json:"connection_uuid" mapstructure:"connection_uuid" validate:"required"`
// 	IPAddress      string `gorm:"column:ip_address" json:"ip_address" mapstructure:"id_address" validate:"required"`
// 	Data           Data   `gorm:"column:data" json:"data" mapstructure:"data" validate:"required"`
// }

// type Data struct {
// 	Bench []Bench `json:"info" mapstructure:"info" validate:"required"`
// }

// type Bench struct {
// 	Type    string `json:"type" mapstructure:"type" validate:"required"`
// 	Desc    string `json:"desc" mapstructure:"desc" validate:"required"`
// 	Elapsed string `json:"elapsed" mapstructure:"elapsed" validate:"required"`
// 	Result  string `json:"result" mapstructure:"result" validate:"required"`
// 	SpecID  string `json:"specid" mapstructure:"specid" validate:"required"`
// 	Unit    string `json:"unit" mapstructure:"unit" validate:"required"`
// }

// func (d Data) Value() (driver.Value, error) {
// 	return json.Marshal(d)
// }

// func (d *Data) Scan(value interface{}) error {
// 	if value == nil {
// 		return nil
// 	}
// 	bytes, ok := value.([]byte)
// 	if !ok {
// 		return errors.New("Invalid type for Data")
// 	}
// 	return json.Unmarshal(bytes, d)
// }
