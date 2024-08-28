package software

import (
	"bufio"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"io"
	"os"
	"strconv"
	"strings"
)

func parseLine(line string) (string, string) {
	// returns (key, value) or ("", value) if multi-line value
	line = strings.TrimRight(line, "\n")

	if len(line) == 0 {
		return "", ""
	}

	if line[0] == ' ' {
		return "", line
	}

	separatorIndex := strings.Index(line, ":")
	key := line[0:separatorIndex]
	value := line[separatorIndex+1:]

	return key, value
}

func mapToDEB(m map[string]string) (software.DEB, error) {
	pkg := software.DEB{}

	for key, value := range m {
		value = strings.TrimRight(value, " \n")
		value = strings.TrimLeft(value, " ")

		switch key {
		case "Package":
			pkg.Package = value
		case "Version":
			pkg.Version = value
		case "Section":
			pkg.Section = value
		case "Installed-Size":
			i, err := strconv.Atoi(value)
			if err == nil {
				pkg.InstalledSize = int64(i)
			}
		case "Maintainer":
			pkg.Maintainer = value
		case "Status":
			pkg.Status = value
		case "Source":
			pkg.Source = value
		case "Architecture":
			pkg.Architecture = value
		case "Multi-Arch":
			pkg.MultiArch = value
		case "Depends":
			pkg.Depends = value
		case "Pre-Depends":
			pkg.PreDepends = value
		case "Description":
			pkg.Description = value
		case "Homepage":
			pkg.Homepage = value
		case "Priority":
			pkg.Priority = value
		}
	}

	return pkg, nil
}

func parse(rd io.Reader) []software.DEB {
	prevKey := ""
	var packages []software.DEB
	m := make(map[string]string)

	for {
		line, readError := bufio.NewReader(rd).ReadString('\n')
		key, value := parseLine(line)

		if key == "" && value != "" {
			m[prevKey] = m[prevKey] + value
		} else if key == "" {
			if len(m) > 0 {
				pkg, err := mapToDEB(m)
				if err == nil {
					packages = append(packages, pkg)
				}
				m = make(map[string]string)
			}
		} else if value != "" {
			prevKey = key
			m[key] = value
		}

		if readError != nil {
			if len(m) > 0 {
				pkg, err := mapToDEB(m)
				if err == nil {
					packages = append(packages, pkg)
				}
			}
			break
		}
	}

	return packages
}

func getConfigFiles(packageName string) ([]string, error) {
	var configs []string

	conffiles := "/var/lib/dpkg/info/" + packageName + ".conffiles"

	if !fileutil.IsExist(conffiles) {
		return []string{}, nil
	}

	fd, err := os.Open(conffiles)
	if err != nil {
		return []string{}, err
	}
	defer func() {
		_ = fd.Close()
	}()

	var line string
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line = scanner.Text()
		line = strings.TrimSpace(line)
		configs = append(configs, line)
	}

	if err := scanner.Err(); err != nil {
		return []string{}, err
	}

	return configs, nil
}

func GetDEBs(showDefaultPackages bool) ([]software.DEB, error) {
	var DEBs []software.DEB

	dpkgStatusFile := "/var/lib/dpkg/status"

	fd, err := os.Open(dpkgStatusFile)
	if err != nil {
		return []software.DEB{}, err
	}
	defer func() {
		_ = fd.Close()
	}()

	rd := bufio.NewReader(fd)
	DEBs = parse(rd)

	for i := range DEBs {
		packageName := DEBs[i].Package
		configs, err := getConfigFiles(DEBs[i].Package)
		if err != nil {
			logger.Println(logger.DEBUG, false, "DEB: Error occurred while reading conffiles of '"+
				packageName+"' package.")
		}
		DEBs[i].Conffiles = configs
	}

	if !showDefaultPackages {
		var filteredDEBs []software.DEB

		defaultPackages, err := GetDefaultPackages()
		if err != nil {
			logger.Println(logger.DEBUG, false, "DEB: Error occurred while getting default packages."+
				" ("+err.Error()+")")
		}

		for _, deb := range DEBs {
			var defPkgFound bool

			for _, defPkg := range defaultPackages {
				if defPkg == deb.Package {
					defPkgFound = true
					break
				}
			}

			if defPkgFound {
				continue
			}

			filteredDEBs = append(filteredDEBs, deb)
		}

		return filteredDEBs, nil
	}

	return DEBs, nil
}
