package dao

import (
	"errors"

	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"gorm.io/gorm"
)

func SavedNLBInfoGet(connectionID string) (*model.SavedNLBInfo, error) {
	savedNLBInfo := &model.SavedNLBInfo{}

	result := db.DB.Where("connection_id = ?", connectionID).First(savedNLBInfo)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SavedNLBInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return savedNLBInfo, nil
}

func SavedNLBInfoRegister(savedNLBInfo *model.SavedNLBInfo) (*model.SavedNLBInfo, error) {
	result := db.DB.Create(savedNLBInfo)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return savedNLBInfo, nil
}

func SavedNLBInfoUpdate(savedNLBInfo *model.SavedNLBInfo) error {
	result := db.DB.Model(&model.SavedNLBInfo{}).Where("connection_id = ?", savedNLBInfo.ConnectionID).Updates(savedNLBInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedNLBInfoDelete(savedNLBInfo *model.SavedNLBInfo) error {
	result := db.DB.Delete(savedNLBInfo)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
