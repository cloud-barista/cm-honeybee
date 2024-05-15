package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/db"
	"github.com/cloud-barista/cm-honeybee/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func MigrationGroupRegister(migrationGroup *onprem.MigrationGroup) (*onprem.MigrationGroup, error) {
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

func MigrationGroupGetList(migrationGroup *onprem.MigrationGroup, page int, row int) (*[]onprem.MigrationGroup, error) {
	migrationGroups := &[]onprem.MigrationGroup{}

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

func MigrationGroupUpdate(migrationGroup *onprem.MigrationGroup) error {
	result := db.DB.Model(&onprem.MigrationGroup{}).Where("uuid = ?", migrationGroup.UUID).Updates(migrationGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func MigrationGroupDelete(migrationGroup *onprem.MigrationGroup) error {
	result := db.DB.Delete(migrationGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func MigrationGroupCheckConnection(migrationGroup *onprem.MigrationGroup) (*[]onprem.ConnectionInfo, error) {
	connectionInfoList, err := ConnectionInfoGetList(&onprem.ConnectionInfo{GroupUUID: migrationGroup.UUID}, 0, 0)
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

	return ConnectionInfoGetList(&onprem.ConnectionInfo{GroupUUID: migrationGroup.UUID}, 0, 0)
}
