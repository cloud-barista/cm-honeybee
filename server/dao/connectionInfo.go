package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"gorm.io/gorm"
	"strconv"
)

func ConnectionInfoRegister(connectionInfo *model.ConnectionInfo) (*model.ConnectionInfo, error) {
	result := db.DB.Create(connectionInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return connectionInfo, nil
}

func ConnectionInfoGet(ID string) (*model.ConnectionInfo, error) {
	connectionInfo := &model.ConnectionInfo{}

	result := db.DB.Where("id = ?", ID).First(connectionInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("ConnectionInfo not found with the provided ID")
		}
		return nil, err
	}

	return connectionInfo, nil
}

func ConnectionInfoGetList(connectionInfo *model.ConnectionInfo, page int, row int) (*[]model.ConnectionInfo, error) {
	connectionInfos := &[]model.ConnectionInfo{}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(connectionInfo.Name) != 0 {
			filtered = filtered.Where("name LIKE ?", "%"+connectionInfo.Name+"%")
		}

		if len(connectionInfo.Description) != 0 {
			filtered = filtered.Where("description LIKE ?", "%"+connectionInfo.Description+"%")
		}

		filtered = filtered.Where("source_group_id LIKE ?", "%"+connectionInfo.SourceGroupID+"%")

		if len(connectionInfo.IPAddress) != 0 {
			filtered = filtered.Where("ip_address LIKE ?", "%"+connectionInfo.IPAddress+"%")
		}

		sshPort, _ := strconv.Atoi(connectionInfo.SSHPort)
		if sshPort >= 1 && sshPort <= 65535 {
			filtered = filtered.Where("ssh_port = ?", "%"+connectionInfo.SSHPort+"%")
		}

		if len(connectionInfo.User) != 0 {
			filtered = filtered.Where("user LIKE ?", "%"+connectionInfo.User+"%")
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
	result := db.DB.Model(&model.ConnectionInfo{}).Where("id = ?", connectionInfo.ID).Updates(connectionInfo)
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
