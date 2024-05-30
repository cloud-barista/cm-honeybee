package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SourceGroupRegister(sourceGroup *model.SourceGroup) (*model.SourceGroup, error) {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	sourceGroup.UUID = UUID.String()

	result := db.DB.Create(sourceGroup)
	err = result.Error
	if err != nil {
		return nil, err
	}

	return sourceGroup, nil
}

func SourceGroupGet(UUID string) (*model.SourceGroup, error) {
	sourceGroup := &model.SourceGroup{}

	result := db.DB.Where("uuid = ?", UUID).First(sourceGroup)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SourceGroup not found with the provided UUID")
		}
		return nil, err
	}

	return sourceGroup, nil
}

func SourceGroupGetList(sourceGroup *model.SourceGroup, page int, row int) (*[]model.SourceGroup, error) {
	sourceGroups := &[]model.SourceGroup{}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(sourceGroup.UUID) != 0 {
			filtered = filtered.Where("uuid LIKE ?", "%"+sourceGroup.UUID+"%")
		}

		if len(sourceGroup.Name) != 0 {
			filtered = filtered.Where("group_uuid LIKE ?", "%"+sourceGroup.Name+"%")
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
	}).Find(sourceGroups)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return sourceGroups, nil
}

func SourceGroupUpdate(sourceGroup *model.SourceGroup) error {
	result := db.DB.Model(&model.SourceGroup{}).Where("uuid = ?", sourceGroup.UUID).Updates(sourceGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SourceGroupDelete(sourceGroup *model.SourceGroup) error {
	result := db.DB.Delete(sourceGroup)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SourceGroupCheckConnection(sourceGroup *model.SourceGroup) (*[]model.ConnectionInfo, error) {
	connectionInfoList, err := ConnectionInfoGetList(&model.ConnectionInfo{GroupUUID: sourceGroup.UUID}, 0, 0)
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

	return ConnectionInfoGetList(&model.ConnectionInfo{GroupUUID: sourceGroup.UUID}, 0, 0)
}
