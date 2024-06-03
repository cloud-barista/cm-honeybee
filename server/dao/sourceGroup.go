package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"gorm.io/gorm"
)

func SourceGroupRegister(sourceGroup *model.SourceGroup) (*model.SourceGroup, error) {
	result := db.DB.Create(sourceGroup)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return sourceGroup, nil
}

func SourceGroupGet(ID string) (*model.SourceGroup, error) {
	sourceGroup := &model.SourceGroup{}

	result := db.DB.Where("id = ?", ID).First(sourceGroup)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SourceGroup not found with the provided ID")
		}
		return nil, err
	}

	return sourceGroup, nil
}

func SourceGroupGetList(sourceGroup *model.SourceGroup, page int, row int) (*[]model.SourceGroup, error) {
	sourceGroups := &[]model.SourceGroup{}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(sourceGroup.ID) != 0 {
			filtered = filtered.Where("id LIKE ?", "%"+sourceGroup.ID+"%")
		}

		if len(sourceGroup.Description) != 0 {
			filtered = filtered.Where("description LIKE ?", "%"+sourceGroup.Description+"%")
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
	result := db.DB.Model(&model.SourceGroup{}).Where("id = ?", sourceGroup.ID).Updates(sourceGroup)
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
	connectionInfoList, err := ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sourceGroup.ID}, 0, 0)
	if err != nil {
		return nil, err
	}

	for _, connectionInfo := range *connectionInfoList {
		oldConnectionInfo, err := ConnectionInfoGet(connectionInfo.ID)
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
				"(ID: " + oldConnectionInfo.ID + ", Error: " + err.Error() + ")")
		}
	}

	return ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sourceGroup.ID}, 0, 0)
}
