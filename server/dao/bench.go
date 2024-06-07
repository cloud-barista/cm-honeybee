package dao

import (
	"errors"

	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"gorm.io/gorm"
)

func SavedBenchmarkInfoRegister(benchmark *model.SavedBenchmarkInfo) (*model.SavedBenchmarkInfo, error) {
	result := db.DB.Create(benchmark)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return benchmark, nil
}

func SavedBenchmarkInfoGet(ConnectionID string) (*model.SavedBenchmarkInfo, error) {
	benchmark := &model.SavedBenchmarkInfo{}

	result := db.DB.Where("connection_id = ?", ConnectionID).First(benchmark)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("BenchmarkInfo not found with the provided connection_id")
		}
		return nil, err
	}

	return benchmark, nil
}

func SavedBenchmarkInfoUpdate(benchmark *model.SavedBenchmarkInfo) error {
	result := db.DB.Model(&model.SavedBenchmarkInfo{}).Where("connection_id = ?", benchmark.ConnectionID).Updates(benchmark)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SavedBenchmarkDelete(benchmark *model.SavedBenchmarkInfo) error {
	result := db.DB.Delete(benchmark)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
