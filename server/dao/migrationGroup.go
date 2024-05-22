package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func MigrationGroupRegister(migrationGroup *model.MigrationGroup) (*model.MigrationGroup, error) {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	migrationGroup.UUID = UUID.String()

	result := db.DB.Create(migrationGroup)
	err = result.Error
	if err != nil {
		return nil, err
	}

	return migrationGroup, nil
}

func MigrationGroupGet(UUID string) (*model.MigrationGroup, error) {
	migrationGroup := &model.MigrationGroup{}

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

func MigrationGroupGetList(migrationGroup *model.MigrationGroup, page int, row int) (*[]model.MigrationGroup, error) {
	migrationGroups := &[]model.MigrationGroup{}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(migrationGroup.UUID) != 0 {
			filtered = filtered.Where("uuid LIKE ?", "%"+migrationGroup.UUID+"%")
		}

		if len(migrationGroup.Name) != 0 {
			filtered = filtered.Where("group_uuid LIKE ?", "%"+migrationGroup.Name+"%")
		}

		if page != 0 && row != 0 {
			offset := (page - 1) * row

			return filtered.Offset(offset).Limit(row)
		} else if row != 0 && page == 0 {
			filtered.Error = errors.New("row is not 0 but page is 0")
			return filtered
		} else if page != 0 && row == 0 {
			filtered.Error = errors.New("page is not 0 but row is 0")
			return filtered
		}

		return filtered
	}).Find(migrationGroups)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return migrationGroups, nil
}

func MigrationGroupUpdate(migrationGroup *model.MigrationGroup) error {
	result := db.DB.Model(&model.MigrationGroup{}).Where("uuid = ?", migrationGroup.UUID).Updates(migrationGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func MigrationGroupDelete(migrationGroup *model.MigrationGroup) error {
	result := db.DB.Delete(migrationGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func MigrationGroupCheckConnection(migrationGroup *model.MigrationGroup) (*[]model.ConnectionInfo, error) {
	connectionInfoList, err := ConnectionInfoGetList(&model.ConnectionInfo{GroupUUID: migrationGroup.UUID}, 0, 0)
	if err != nil {
		return nil, err
	}

	for _, connectionInfo := range *connectionInfoList {
		oldConnectionInfo, err := ConnectionInfoGet(connectionInfo.UUID)
		if err != nil {
			return nil, err
		}

		c := &ssh.SSH{
			Options: ssh.DefaultSSHOptions(),
		}

		err = c.NewClientConn(connectionInfo)
		if err != nil {
			oldConnectionInfo.Status = "Failed"
			oldConnectionInfo.FailedMessage = err.Error()
		}

		if err == nil {
			oldConnectionInfo.Status = "Success"
			oldConnectionInfo.FailedMessage = " " // Can't set empty string.
		}

		err = ConnectionInfoUpdate(oldConnectionInfo)
		if err != nil {
			return nil, errors.New("Error occurred while updating the connection information. " +
				"(UUID: " + oldConnectionInfo.UUID + ", Error: " + err.Error() + ")")
		}
	}

	return ConnectionInfoGetList(&model.ConnectionInfo{GroupUUID: migrationGroup.UUID}, 0, 0)
}
