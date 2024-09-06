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

func SavedInfraInfoGet(connectionID string) (*model.SavedInfraInfo, error) {
	savedInfraInfo := &model.SavedInfraInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedInfraInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedInfraInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedInfraInfo, nil
}

func SavedInfraInfoUpdate(savedInfraInfo *model.SavedInfraInfo) error {
	result := db.DB.Model(&model.SavedInfraInfo{}).Where("connection_id = ?", savedInfraInfo.ConnectionID).Updates(savedInfraInfo)
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

func SavedSoftwareInfoGet(connectionID string) (*model.SavedSoftwareInfo, error) {
	savedSoftwareInfo := &model.SavedSoftwareInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedSoftwareInfo)
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
	result := db.DB.Model(&model.SavedSoftwareInfo{}).Where("connection_id = ?", savedSoftwareInfo.ConnectionID).Updates(savedSoftwareInfo)
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

func SavedKubernetesInfoGet(connectionID string) (*model.SavedKubernetesInfo, error) {
	savedKubernetesInfo := &model.SavedKubernetesInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedKubernetesInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedKubernetesInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedKubernetesInfo, nil
}

func SavedKubernetesInfoRegister(savedKubernetesInfo *model.SavedKubernetesInfo) (*model.SavedKubernetesInfo, error) {
	result := db.DB.Create(savedKubernetesInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedKubernetesInfo, nil
}

func SavedKubernetesInfoUpdate(savedKubernetesInfo *model.SavedKubernetesInfo) error {
	result := db.DB.Model(&model.SavedKubernetesInfo{}).Where("connection_id = ?", savedKubernetesInfo.ConnectionID).Updates(savedKubernetesInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
