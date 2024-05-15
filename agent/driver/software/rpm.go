package software

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	_ "github.com/glebarez/go-sqlite" // sqlite
	"github.com/hashicorp/go-multierror"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
)

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

func GetRPMs() ([]software.RPM, error) {
	db, err := detectDB()
	if err != nil {
		return []software.RPM{}, err
	}
	pkgList, err := db.ListPackages()
	if err != nil {
		return []software.RPM{}, err
	}

	var rpms []software.RPM

	for _, pkg := range pkgList {
		rpms = append(rpms, software.RPM{
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
