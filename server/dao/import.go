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

func SavedKubernetesInfoDelete(savedKubernetesInfo *model.SavedKubernetesInfo) error {
	result := db.DB.Delete(savedKubernetesInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedHelmInfoGet(connectionID string) (*model.SavedHelmInfo, error) {
	savedHelmInfo := &model.SavedHelmInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedHelmInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("savedHelmInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedHelmInfo, nil
}

func SavedHelmInfoRegister(savedHelmInfo *model.SavedHelmInfo) (*model.SavedHelmInfo, error) {
	result := db.DB.Create(savedHelmInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedHelmInfo, nil
}

func SavedHelmInfoUpdate(savedHelmInfo *model.SavedHelmInfo) error {
	result := db.DB.Model(&model.SavedHelmInfo{}).Where("connection_id = ?", savedHelmInfo.ConnectionID).Updates(savedHelmInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedHelmInfoDelete(savedHelmInfo *model.SavedHelmInfo) error {
	result := db.DB.Delete(savedHelmInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedDataInfoGet(connectionID string) (*model.SavedDataInfo, error) {
	savedDataInfo := &model.SavedDataInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedDataInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedDataInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedDataInfo, nil
}

func SavedDataInfoRegister(savedDataInfo *model.SavedDataInfo) (*model.SavedDataInfo, error) {
	result := db.DB.Create(savedDataInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedDataInfo, nil
}

func SavedDataInfoUpdate(savedDataInfo *model.SavedDataInfo) error {
	result := db.DB.Model(&model.SavedDataInfo{}).Where("connection_id = ?", savedDataInfo.ConnectionID).Updates(savedDataInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedDataInfoDelete(savedDataInfo *model.SavedDataInfo) error {
	result := db.DB.Delete(savedDataInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedFSInfoGet(connectionID string) (*model.SavedFSInfo, error) {
	savedFSInfo := &model.SavedFSInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedFSInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedFSInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedFSInfo, nil
}

func SavedFSInfoRegister(savedFSInfo *model.SavedFSInfo) (*model.SavedFSInfo, error) {
	result := db.DB.Create(savedFSInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedFSInfo, nil
}

func SavedFSInfoUpdate(savedFSInfo *model.SavedFSInfo) error {
	result := db.DB.Model(&model.SavedFSInfo{}).Where("connection_id = ?", savedFSInfo.ConnectionID).Updates(savedFSInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedFSInfoDelete(savedFSInfo *model.SavedFSInfo) error {
	result := db.DB.Delete(savedFSInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedObjectStorageInfoGet(connectionID string) (*model.SavedObjectStorageInfo, error) {
	savedObjectStorageInfo := &model.SavedObjectStorageInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedObjectStorageInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedObjectStorageInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedObjectStorageInfo, nil
}

func SavedObjectStorageInfoRegister(savedObjectStorageInfo *model.SavedObjectStorageInfo) (*model.SavedObjectStorageInfo, error) {
	result := db.DB.Create(savedObjectStorageInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedObjectStorageInfo, nil
}

func SavedObjectStorageInfoUpdate(savedObjectStorageInfo *model.SavedObjectStorageInfo) error {
	result := db.DB.Model(&model.SavedObjectStorageInfo{}).Where("connection_id = ?", savedObjectStorageInfo.ConnectionID).Updates(savedObjectStorageInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedObjectStorageInfoDelete(savedObjectStorageInfo *model.SavedObjectStorageInfo) error {
	result := db.DB.Delete(savedObjectStorageInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedDBInfoGet(connectionID string) (*model.SavedDBInfo, error) {
	savedDBInfo := &model.SavedDBInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedDBInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedDBInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedDBInfo, nil
}

func SavedDBInfoRegister(savedDBInfo *model.SavedDBInfo) (*model.SavedDBInfo, error) {
	result := db.DB.Create(savedDBInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedDBInfo, nil
}

func SavedDBInfoUpdate(savedDBInfo *model.SavedDBInfo) error {
	result := db.DB.Model(&model.SavedDBInfo{}).Where("connection_id = ?", savedDBInfo.ConnectionID).Updates(savedDBInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedDBInfoDelete(savedDBInfo *model.SavedDBInfo) error {
	result := db.DB.Delete(savedDBInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
