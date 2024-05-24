package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/db"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strconv"
)

func ConnectionInfoRegister(connectionInfo *model.ConnectionInfo) (*model.ConnectionInfo, error) {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	connectionInfo.UUID = UUID.String()

	result := db.DB.Create(connectionInfo)
	err = result.Error
	if err != nil {
		return nil, err
	}

	return connectionInfo, nil
}

func ConnectionInfoGet(UUID string) (*model.ConnectionInfo, error) {
	connectionInfo := &model.ConnectionInfo{}

	result := db.DB.Where("uuid = ?", UUID).First(connectionInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("ConnectionInfo not found with the provided UUID")
		}
		return nil, err
	}

	return connectionInfo, nil
}

func ConnectionInfoGetList(connectionInfo *model.ConnectionInfo, page int, row int) (*[]model.ConnectionInfo, error) {
	connectionInfos := &[]model.ConnectionInfo{}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(connectionInfo.UUID) != 0 {
			filtered = filtered.Where("uuid LIKE ?", "%"+connectionInfo.UUID+"%")
		}

		if len(connectionInfo.GroupUUID) != 0 {
			filtered = filtered.Where("group_uuid LIKE ?", "%"+connectionInfo.GroupUUID+"%")
		}

		if len(connectionInfo.IPAddress) != 0 {
			filtered = filtered.Where("ip_address LIKE ?", "%"+connectionInfo.IPAddress+"%")
		}

		if connectionInfo.SSHPort >= 1 && connectionInfo.SSHPort <= 65535 {
			filtered = filtered.Where("ssh_port = ?", "%"+strconv.Itoa(connectionInfo.SSHPort)+"%")
		}

		if len(connectionInfo.User) != 0 {
			filtered = filtered.Where("user LIKE ?", "%"+connectionInfo.User+"%")
		}

		if len(connectionInfo.Type) != 0 {
			filtered = filtered.Where("type LIKE ?", "%"+connectionInfo.Type+"%")
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
	}).Find(connectionInfos)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return connectionInfos, nil
}

func ConnectionInfoUpdate(connectionInfo *model.ConnectionInfo) error {
	result := db.DB.Model(&model.ConnectionInfo{}).Where("uuid = ?", connectionInfo.UUID).Updates(connectionInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func ConnectionInfoDelete(connectionInfo *model.ConnectionInfo) error {
	result := db.DB.Delete(connectionInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
