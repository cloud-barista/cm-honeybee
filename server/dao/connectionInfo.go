package dao

import (
	"errors"
	"strconv"

	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"gorm.io/gorm"
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
	return ConnectionInfoGetListOfTypes(connectionInfo, nil, page, row)
}

// ConnectionInfoGetListOfTypes is ConnectionInfoGetList narrowed to the
// connections whose stored resource_type is one of resourceTypes. A nil
// resourceTypes does not narrow. The caller passes the stored values, since
// one type may be stored under more than one value (an on-prem VM registered
// before resource_type existed has an empty one).
func ConnectionInfoGetListOfTypes(connectionInfo *model.ConnectionInfo, resourceTypes []string, page int, row int) (*[]model.ConnectionInfo, error) {
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
			filtered = filtered.Where("ssh_port = ?", strconv.Itoa(sshPort))
		}

		if resourceTypes != nil {
			filtered = filtered.Where("resource_type IN ?", resourceTypes)
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
	}).Order("name").Find(connectionInfos)

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

func ConnectionInfoUpdateWithSelect(connectionInfo *model.ConnectionInfo, fields []string) error {
	result := db.DB.Select(fields).Where("id = ?", connectionInfo.ID).Updates(connectionInfo)
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

// ConnectionTypeCount counts the connections of one source group that share a
// stored resource_type.
type ConnectionTypeCount struct {
	SourceGroupID string `gorm:"column:source_group_id"`
	ResourceType  string `gorm:"column:resource_type"`
	Total         int    `gorm:"column:total"`
	// ConnectionSuccess and AgentSuccess count the connections whose
	// connection_status or agent_status is success.
	ConnectionSuccess int `gorm:"column:connection_success"`
	AgentSuccess      int `gorm:"column:agent_success"`
	// WithIPAddress counts the connections that have an ip_address.
	WithIPAddress int `gorm:"column:with_ip_address"`
}

// ConnectionInfoCountByType counts the connections of the given source groups
// by source group and stored resource_type in one query, so a group list does
// not read the connections of each group on its own.
func ConnectionInfoCountByType(sourceGroupIDs []string) ([]ConnectionTypeCount, error) {
	counts := []ConnectionTypeCount{}
	if len(sourceGroupIDs) == 0 {
		return counts, nil
	}

	result := db.DB.Model(&model.ConnectionInfo{}).
		Select("source_group_id, COALESCE(resource_type, '') AS resource_type, COUNT(*) AS total, "+
			"SUM(CASE WHEN connection_status = ? THEN 1 ELSE 0 END) AS connection_success, "+
			"SUM(CASE WHEN agent_status = ? THEN 1 ELSE 0 END) AS agent_success, "+
			"SUM(CASE WHEN TRIM(COALESCE(ip_address, '')) <> '' THEN 1 ELSE 0 END) AS with_ip_address",
			model.ConnectionInfoStatusSuccess, model.ConnectionInfoStatusSuccess).
		Where("source_group_id IN ?", sourceGroupIDs).
		Group("source_group_id, COALESCE(resource_type, '')").
		Order("source_group_id, resource_type").
		Find(&counts)
	if result.Error != nil {
		return nil, result.Error
	}

	return counts, nil
}
