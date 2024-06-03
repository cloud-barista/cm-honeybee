package model

type SourceGroup struct {
	ID          string `gorm:"primaryKey" json:"id" validate:"required"`
	Description string `gorm:"column:description" json:"description"`
}
