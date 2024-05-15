package onprem

type MigrationGroup struct {
	UUID string `gorm:"primaryKey" json:"uuid" validate:"required"`
	Name string `orm:"column:name" json:"name" validate:"required"`
}
