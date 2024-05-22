package dao

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"gorm.io/gorm"
)

func SavedInfraInfoRegister(savedInfraInfo *model.SavedInfraInfo) (*model.SavedInfraInfo, error) {
	result := db.DB.Create(savedInfraInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedInfraInfo, nil
}

func SavedInfraInfoGet(ConnectionUUID string) (*model.SavedInfraInfo, error) {
	savedInfraInfo := &model.SavedInfraInfo{}

	result := db.DB.Where("connection_uuid = ?", ConnectionUUID).First(savedInfraInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedInfraInfo not found with the provided connection_uuid")
		}
		return nil, err
	}

	return savedInfraInfo, nil
}

func SavedInfraInfoUpdate(savedInfraInfo *model.SavedInfraInfo) error {
	result := db.DB.Model(&model.SavedInfraInfo{}).Where("connection_uuid = ?", savedInfraInfo.ConnectionUUID).Updates(savedInfraInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedInfraInfoDelete(savedInfraInfo *model.SavedInfraInfo) error {
	result := db.DB.Delete(savedInfraInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedSoftwareInfoRegister(savedSoftwareInfo *model.SavedSoftwareInfo) (*model.SavedSoftwareInfo, error) {
	result := db.DB.Create(savedSoftwareInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedSoftwareInfo, nil
}

func SavedSoftwareInfoGet(ConnectionUUID string) (*model.SavedSoftwareInfo, error) {
	savedSoftwareInfo := &model.SavedSoftwareInfo{}

	result := db.DB.Where("connection_uuid = ?", ConnectionUUID).First(savedSoftwareInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedSoftwareInfo not found with the provided connection_uuid")
		}
		return nil, err
	}

	return savedSoftwareInfo, nil
}

func SavedSoftwareInfoUpdate(savedSoftwareInfo *model.SavedSoftwareInfo) error {
	result := db.DB.Model(&model.SavedSoftwareInfo{}).Where("connection_uuid = ?", savedSoftwareInfo.ConnectionUUID).Updates(savedSoftwareInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedSoftwareInfoDelete(savedSoftwareInfo *model.SavedSoftwareInfo) error {
	result := db.DB.Delete(savedSoftwareInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
