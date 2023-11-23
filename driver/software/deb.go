package software

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

type DEB struct {
	Package       string `json:"package"`
	Status        string `json:"status"`
	Priority      string `json:"priority"`
	Architecture  string `json:"architecture"`
	MultiArch     string `json:"multi_arch"`
	Maintainer    string `json:"maintainer"`
	Version       string `json:"version"`
	Section       string `json:"section"`
	InstalledSize int64  `json:"installed_size"`
	Depends       string `json:"depends"`
	PreDepends    string `json:"pre_depends"`
	Description   string `json:"description"`
	Source        string `json:"source"`
	Homepage      string `json:"homepage"`
}

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

func mapToDEB(m map[string]string) (DEB, error) {
	pkg := DEB{}

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

func parse(rd io.Reader) []DEB {
	prevKey := ""
	var packages []DEB
	m := make(map[string]string)

	for {
		line, readError := bufio.NewReader(rd).ReadString('\n')
		key, value := parseLine(line)

		if key == "" && value != "" {
			m[prevKey] = m[prevKey] + value
		} else if key == "" && value == "" {
			if len(m) > 0 {
				pkg, err := mapToDEB(m)
				if err == nil {
					packages = append(packages, pkg)
				}
				m = make(map[string]string)
			}
		} else if key != "" && value != "" {
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

func GetDEBs() ([]DEB, error) {
	dpkgStatusFile := "/var/lib/dpkg/status"

	fd, err := os.Open(dpkgStatusFile)
	if err != nil {
		return []DEB{}, err
	}
	defer func() {
		_ = fd.Close()
	}()

	rd := bufio.NewReader(fd)
	return parse(rd), nil
}
