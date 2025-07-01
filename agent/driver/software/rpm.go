package software

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	_ "github.com/glebarez/go-sqlite" // sqlite
	"github.com/hashicorp/go-multierror"
	"github.com/jollaman999/utils/logger"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"strings"
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

func isValidPackageName(name string) bool {
	if name == "" {
		return false
	}

	if strings.ContainsAny(name, "=<>()[]{}") {
		return false
	}

	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') && r != '-' && r != '_' && r != '.' && r != '+' {
			return false
		}
	}

	hasAlpha := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasAlpha = true
			break
		}
	}

	return hasAlpha
}

func extractPackageName(req string) string {
	req = strings.TrimSpace(req)

	if strings.HasPrefix(req, "rpmlib(") {
		return ""
	}

	if strings.HasPrefix(req, "config(") {
		return ""
	}

	if strings.HasPrefix(req, "/") {
		return ""
	}

	if strings.HasPrefix(req, "lib") && strings.Contains(req, ".so") {
		return ""
	}

	if strings.HasPrefix(req, "rtld(") {
		return ""
	}

	if parenIndex := strings.Index(req, "("); parenIndex != -1 {
		packageName := strings.TrimSpace(req[:parenIndex])

		if isValidPackageName(packageName) {
			return packageName
		}
		return ""
	}

	if isValidPackageName(req) {
		return req
	}

	return ""
}

func parseRpmRequires(requires []string) []string {
	var packages []string
	packageSet := make(map[string]bool)

	for _, req := range requires {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}

		packageName := extractPackageName(req)
		if packageName != "" && !packageSet[packageName] {
			packages = append(packages, packageName)
			packageSet[packageName] = true
		}
	}

	return packages
}

func GetRPMs(showDefaultPackages bool) ([]software.RPM, error) {
	db, err := detectDB()
	if err != nil {
		return []software.RPM{}, err
	}
	defer func() {
		_ = db.Close()
	}()
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
			Group:     pkg.Group,
			Arch:      pkg.Arch,
			SourceRpm: pkg.SourceRpm,
			Size:      pkg.Size,
			License:   pkg.License,
			Vendor:    pkg.Vendor,
			Summary:   pkg.Summary,
			Requires:  pkg.Requires,
		})
	}

	var requiresPackages []string
	var requiresRemovedList = make([]software.RPM, 0)

	for _, rpm := range rpms {
		requiresPackages = append(requiresPackages, parseRpmRequires(rpm.Requires)...)
	}

	if showDefaultPackages {
		for _, rpm := range rpms {
			var requirePkgFound bool

			for _, pkg := range requiresPackages {
				if rpm.Name == pkg {
					requirePkgFound = true
					break
				}
			}

			if requirePkgFound {
				continue
			}

			requiresRemovedList = append(requiresRemovedList, rpm)
		}
	} else {
		defaultPackages, err := GetDefaultPackages()
		if err != nil {
			logger.Println(logger.DEBUG, false, "RPM: Error occurred while getting default packages."+
				" ("+err.Error()+")")
		}

		for _, rpm := range rpms {
			var defPkgFound bool

			for _, defPkg := range defaultPackages {
				if defPkg == rpm.Name {
					defPkgFound = true
					break
				}
			}

			if defPkgFound {
				continue
			}

			var depPkgFound bool

			for _, pkg := range requiresPackages {
				if rpm.Name == pkg {
					depPkgFound = true
					break
				}
			}

			if depPkgFound {
				continue
			}

			requiresRemovedList = append(requiresRemovedList, rpm)
		}
	}

	return rpms, nil
}
