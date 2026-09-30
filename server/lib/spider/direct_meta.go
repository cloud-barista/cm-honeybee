package spider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	"gopkg.in/yaml.v3"
)

// ListCloudOS returns the supported CSP names from cloudos.yaml.
func ListCloudOS() ([]string, error) {
	if err := cbspiderFile(filepath.Join("cloud-driver-libs", "cloudos.yaml")); err != nil {
		return nil, err
	}
	return cim.ListCloudOS(), nil
}

// GetCloudOSMetaInfo returns the metadata for a Cloud OS from
// cloudos_meta.yaml. An unknown name yields empty metadata, as the REST
// /cloudos/metainfo does.
func GetCloudOSMetaInfo(cloudOSName string) (*CloudOSMetaInfo, error) {
	if err := mustNonEmpty("CloudOSName", cloudOSName); err != nil {
		return nil, err
	}
	meta, err := loadMetaInfo(cloudOSName)
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[CloudOSMetaInfo](meta)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDriverCapability reports which handlers the provider's driver
// implements. No connection is needed.
func GetDriverCapability(provider string) (*DriverCapability, error) {
	drv, err := cloudDriver(provider)
	if err != nil {
		return nil, err
	}
	out, err := convertJSON[DriverCapability](drv.GetDriverCapability())
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRegionZone lists the CSP's regions and zones with a credential
// only, as cb-spider's GET /preconfig/regionzone does: the region to query comes from the
// provider's DefaultRegionToQuery.
func ListRegionZone(provider string, credential []KeyValue) ([]RegionZoneInfo, error) {
	p, err := upperProvider(provider)
	if err != nil {
		return nil, err
	}
	if _, err := cloudDriver(p); err != nil {
		return nil, err
	}
	meta, err := loadMetaInfo(p)
	if err != nil {
		return nil, err
	}
	info := queryConnectionInfo(meta, credential)

	conn, err := connectInfo(p, info)
	if err != nil {
		return nil, err
	}
	handler, err := conn.CreateRegionZoneHandler()
	if err != nil {
		return nil, err
	}
	list, err := handler.ListRegionZone()
	if err != nil && len(list) == 0 {
		return nil, err
	}

	list, err = updateRegionZoneDisplayNames(strings.ToLower(p), list)
	if err != nil {
		return nil, err
	}
	return convertList[RegionZoneInfo](list)
}

func queryConnectionInfo(meta cim.CloudOSMetaInfo, credential []KeyValue) idrv.ConnectionInfo {
	info := idrv.ConnectionInfo{CredentialInfo: credentialInfo(mapCredentialKeys(meta, credential))}
	switch len(meta.DefaultRegionToQuery) {
	case 1:
		info.RegionInfo.Region = meta.DefaultRegionToQuery[0]
	case 2:
		info.RegionInfo.Region = meta.DefaultRegionToQuery[0]
		info.RegionInfo.Zone = meta.DefaultRegionToQuery[1]
	}
	return info
}

// updateRegionZoneDisplayNames mirrors cb-spider's
// UpdateRegionZoneDisplayNames(): names come from
// $CBSPIDER_ROOT/cloud-driver-libs/region/<csp>_region_meta.yaml, and a
// missing file leaves the list unchanged.
func updateRegionZoneDisplayNames(csp string, list []*cres.RegionZoneInfo) ([]*cres.RegionZoneInfo, error) {
	metaFile := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "cloud-driver-libs", "region", fmt.Sprintf("%s_region_meta.yaml", csp))
	data, err := os.ReadFile(metaFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return list, nil
		}
		return nil, err
	}

	var metadata struct {
		DisplayName    map[string]map[string]string `yaml:"DisplayName"`
		CSPDisplayName map[string]map[string]string `yaml:"CSPDisplayName"`
	}
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata file: %v", err)
	}

	for _, region := range list {
		if region == nil {
			continue
		}
		if names, ok := metadata.DisplayName[region.Name]; ok {
			if name, ok := names[""]; ok {
				region.DisplayName = name
			}
		}
		if names, ok := metadata.CSPDisplayName[region.Name]; ok {
			if name, ok := names[""]; ok {
				region.CSPDisplayName = name
			}
		}
		for i, zone := range region.ZoneList {
			if name, ok := metadata.DisplayName[region.Name][zone.Name]; ok {
				region.ZoneList[i].DisplayName = name
			}
			if name, ok := metadata.CSPDisplayName[region.Name][zone.Name]; ok {
				region.ZoneList[i].CSPDisplayName = name
			}
		}
	}
	return list, nil
}
