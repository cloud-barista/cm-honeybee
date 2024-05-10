package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/db"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func MigrationGroupRegister(MigrationGroup *onprem.MigrationGroup) (*onprem.MigrationGroup, error) {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	MigrationGroup.UUID = UUID.String()

	result := db.DB.Create(MigrationGroup)
	err = result.Error
	if err != nil {
		return nil, err
	}

	return MigrationGroup, nil
}

func MigrationGroupGet(UUID string) (*onprem.MigrationGroup, error) {
	migrationGroup := &onprem.MigrationGroup{}

	result := db.DB.Where("uuid = ?", UUID).First(migrationGroup)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("MigrationGroup not found with the provided UUID")
		}
		return nil, err
	}

	return migrationGroup, nil
}

func MigrationGroupGetList(MigrationGroup *onprem.MigrationGroup, page int, row int) (*[]onprem.MigrationGroup, error) {
	migrationGroups := &[]onprem.MigrationGroup{}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(MigrationGroup.UUID) != 0 {
			filtered = filtered.Where("uuid LIKE ?", "%"+MigrationGroup.UUID+"%")
		}

		if len(MigrationGroup.Name) != 0 {
			filtered = filtered.Where("group_uuid LIKE ?", "%"+MigrationGroup.Name+"%")
		}

		if page != 0 && row != 0 {
			offset := (page - 1) * row

			return filtered.Offset(offset).Limit(row)
		} else if row != 0 && page == 0 {
			filtered.Error = errors.New("row is not 0 but page is 0")

			return nil
		} else if page != 0 && row == 0 {
			filtered.Error = errors.New("page is not 0 but row is 0")

			return nil
		}

		return filtered
	}).Find(migrationGroups)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return migrationGroups, nil
}

func MigrationGroupUpdate(MigrationGroup *onprem.MigrationGroup) error {
	result := db.DB.Model(&onprem.MigrationGroup{}).Where("uuid = ?", MigrationGroup.UUID).Updates(MigrationGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func MigrationGroupDelete(MigrationGroup *onprem.MigrationGroup) error {
	result := db.DB.Delete(MigrationGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
