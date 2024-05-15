package config

func PrepareConfigs() error {
	err := prepareCMHoneybeeAgentConfig()
	if err != nil {
		return err
	}

	return nil
}
