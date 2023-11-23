package software

type Software struct {
	DEB []DEB `json:"deb"`
}

func GetSoftwareInfo() (*Software, error) {
	deb, err := GetDEBs()
	if err != nil {
		return nil, err
	}

	software := Software{DEB: deb}

	return &software, nil
}
