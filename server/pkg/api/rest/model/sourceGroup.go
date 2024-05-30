package model

type SourceGroup struct {
	UUID string `gorm:"primaryKey" json:"uuid" validate:"required"`
	Name string `orm:"column:name" json:"name" validate:"required"`
}
