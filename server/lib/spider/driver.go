package spider

import "strings"

// CloudDriverInfo mirrors spider.cim.CloudDriverInfo.
type CloudDriverInfo struct {
	DriverName        string `json:"DriverName"`
	ProviderName      string `json:"ProviderName"`
	DriverLibFileName string `json:"DriverLibFileName"`
}

type listDriverResp struct {
	Driver []CloudDriverInfo `json:"driver"`
}

// ListDrivers returns drivers registered with cb-spider, optionally filtered by provider.
func ListDrivers(provider string) ([]CloudDriverInfo, error) {
	path := "/driver"
	if provider != "" {
		path += "?provider=" + encodePath(provider)
	}
	var out listDriverResp
	if err := do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Driver, nil
}

// RegisterDriver registers a Cloud Driver entry.
func RegisterDriver(info CloudDriverInfo) (*CloudDriverInfo, error) {
	if err := mustNonEmpty("DriverName", info.DriverName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("ProviderName", info.ProviderName); err != nil {
		return nil, err
	}
	if err := mustNonEmpty("DriverLibFileName", info.DriverLibFileName); err != nil {
		return nil, err
	}
	var out CloudDriverInfo
	if err := do("POST", "/driver", info, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EnsureDriver returns a usable DriverName for the given provider:
// reuses the first registered driver if present; otherwise registers a new
// driver using the convention "<PROVIDER>-Driver-V1.0" pointing at
// "<lower>-driver-v1.0.so".
func EnsureDriver(provider string) (string, error) {
	if err := mustNonEmpty("ProviderName", provider); err != nil {
		return "", err
	}
	drivers, err := ListDrivers(provider)
	if err != nil {
		return "", err
	}
	for _, d := range drivers {
		if strings.EqualFold(d.ProviderName, provider) && d.DriverName != "" {
			return d.DriverName, nil
		}
	}
	// Fallback to convention-based registration.
	driverName := strings.ToUpper(provider) + "-Driver-V1.0"
	libFile := strings.ToLower(provider) + "-driver-v1.0.so"
	created, err := RegisterDriver(CloudDriverInfo{
		DriverName:        driverName,
		ProviderName:      strings.ToUpper(provider),
		DriverLibFileName: libFile,
	})
	if err != nil {
		return "", err
	}
	return created.DriverName, nil
}
