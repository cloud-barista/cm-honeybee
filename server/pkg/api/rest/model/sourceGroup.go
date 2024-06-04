package model

type SourceGroup struct {
	ID          string `gorm:"primaryKey" json:"id" validate:"required"`
	Name        string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description string `gorm:"column:description" json:"description"`
}

type CreateSourceGroupReq struct {
	Name        string `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" mapstructure:"name" validate:"required"`
	Description string `gorm:"column:description" json:"description"`
}
