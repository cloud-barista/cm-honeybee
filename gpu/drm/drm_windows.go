// Getting DRM information is only available on Linux & Unix like systems

//go:build windows

package drm

type DRM struct {
	DriverName        string `json:"driver_name"`
	DriverVersion     string `json:"driver_version"`
	DriverDate        string `json:"driver_date"`
	DriverDescription string `json:"driver_description"`
}

func GetDRMInfo() ([]DRM, error) {
	var d []DRM

	d = append(d, DRM{
		DriverName:        "N/A",
		DriverVersion:     "N/A",
		DriverDate:        "N/A",
		DriverDescription: "N/A",
	})

	return d, nil
}
