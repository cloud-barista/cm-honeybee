package nvidia

import (
	"github.com/jollaman999/utils/logger"
)

func NewNVReader() (*NVReader, error) {
	var reader NVReader

	if err := reader.Init(); err != nil {
		return nil, err
	}

	logger.Println(logger.INFO, false, "NVML is initialized")

	return &reader, nil
}
