package common

import (
	"github.com/google/uuid"
	"github.com/jollaman999/utils/fileutil"
)

var AgentUUID string

func InitAgentUUID() error {
	var UUID uuid.UUID
	var err error

	uuidFile := RootPath + "/uuid"

	if !fileutil.IsExist(uuidFile) {
		UUID, err = uuid.NewRandom()
		if err != nil {
			return err
		}

		err = fileutil.WriteFile(uuidFile, UUID.String())
		if err != nil {
			return err
		}
	}

	AgentUUID, err = fileutil.ReadFile(uuidFile)
	if err != nil {
		return err
	}

	_, err = uuid.Parse(AgentUUID)
	if err != nil {
		return err
	}

	return nil
}
