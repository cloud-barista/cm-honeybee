package software

import (
	_ "github.com/glebarez/go-sqlite" // sqlite
	"github.com/hashicorp/go-multierror"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
)

type RPM struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Release   string   `json:"release"`
	Arch      string   `json:"arch"`
	SourceRpm string   `json:"sourceRpm"`
	Size      int      `json:"size"`
	License   string   `json:"license"`
	Vendor    string   `json:"vendor"`
	Summary   string   `json:"summary"`
	Requires  []string `json:"requires"`
}

func detectDB() (*rpmdb.RpmDB, error) {
	var result error
	db, err := rpmdb.Open("/var/lib/rpm/rpmdb.sqlite")
	if err == nil {
		return db, nil
	}
	result = multierror.Append(result, err)

	db, err = rpmdb.Open("/var/lib/rpm/Packages.db")
	if err == nil {
		return db, nil
	}
	result = multierror.Append(result, err)

	db, err = rpmdb.Open("/var/lib/rpm/Packages")
	if err == nil {
		return db, nil
	}
	result = multierror.Append(result, err)

	return nil, result
}

func GetRPMs() ([]RPM, error) {
	db, err := detectDB()
	if err != nil {
		return []RPM{}, err
	}
	pkgList, err := db.ListPackages()
	if err != nil {
		return []RPM{}, err
	}

	var rpms []RPM

	for _, pkg := range pkgList {
		rpms = append(rpms, RPM{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Release:   pkg.Release,
			Arch:      pkg.Arch,
			SourceRpm: pkg.SourceRpm,
			Size:      pkg.Size,
			License:   pkg.License,
			Vendor:    pkg.Vendor,
			Summary:   pkg.Summary,
			Requires:  pkg.Requires,
		})
	}

	return rpms, nil
}
