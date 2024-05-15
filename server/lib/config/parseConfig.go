package config

func PrepareConfigs() error {
	err := prepareCMHoneybeeConfig()
	if err != nil {
		return err
	}

	return nil
}
