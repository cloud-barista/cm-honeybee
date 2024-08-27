package software

import (
	"compress/gzip"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/host"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/ulikunitz/xz"
)

// RepoMD represents the structure of repomd.xml for parsing
type RepoMD struct {
	XMLName xml.Name `xml:"repomd"`
	Data    []struct {
		Type     string `xml:"type,attr"`
		Location struct {
			Href string `xml:"href,attr"`
		} `xml:"location"`
	} `xml:"data"`
}

// PackageReq represents a single package requirement in a group
type PackageReq struct {
	Type string `xml:"type,attr"`
	Name string `xml:",chardata"`
}

// Group represents a group in comps.xml.xz
type Group struct {
	ID          string       `xml:"id"`
	Name        string       `xml:"name"`
	PackageList []PackageReq `xml:"packagelist>packagereq"`
}

// Groups represents the structure that contains multiple Group elements
type Groups struct {
	XMLName xml.Name `xml:"comps"`
	Groups  []Group  `xml:"group"`
}

// createTransport creates an HTTP transport that ignores SSL certificate verification
func createTransport() *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

// fetchURL fetches the content from a URL with SSL verification disabled
func fetchURL(url string) ([]byte, error) {
	client := &http.Client{
		Transport: createTransport(),
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("packageFilter: failed to fetch URL %s: %s", url, resp.Status)
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	var reader io.Reader
	reader = resp.Body

	// Check if the file is gzipped or xz compressed
	if strings.HasSuffix(url, ".gz") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = gzipReader.Close()
		}()
		reader = gzipReader
	} else if strings.HasSuffix(url, ".xz") {
		xzReader, err := xz.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		reader = xzReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// parseRepoMD parses the repomd.xml file to find the group_gz or group_xz data location
func parseRepoMD(data []byte) (string, error) {
	var repomd RepoMD
	if err := xml.Unmarshal(data, &repomd); err != nil {
		return "", err
	}

	for _, data := range repomd.Data {
		if data.Type == "group_gz" || data.Type == "group_xz" {
			return data.Location.Href, nil
		}
	}

	errMsg := "no group_gz or group_xz data found in repomd.xml"
	logger.Println(logger.ERROR, true, errMsg)
	return "", errors.New(errMsg)
}

// parseGroups parses the group XML data to extract mandatory and default packages
func parseGroups(data []byte) ([]string, error) {
	var groups Groups
	if err := xml.Unmarshal(data, &groups); err != nil {
		return nil, err
	}

	var packages []string
	for _, group := range groups.Groups {
		// Matching against "core" in a case-insensitive manner
		if strings.ToLower(group.ID) == "core" || strings.ToLower(group.Name) == "core" {
			for _, pkg := range group.PackageList {
				if pkg.Type == "mandatory" || pkg.Type == "default" {
					packages = append(packages, pkg.Name)
				}
			}
			break
		}
	}

	// Sort packages in ascending order
	sort.Strings(packages)

	return packages, nil
}

// parseUbuntuManifest parses the Ubuntu minimal cloud image manifest file to extract package names
func parseUbuntuManifest(data []byte) ([]string, error) {
	lines := strings.Split(string(data), "\n")
	var packages []string

	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) > 1 {
			pkgName := strings.Split(parts[0], ":")[0]
			if pkgName != "" {
				packages = append(packages, pkgName)
			}
		}
	}

	// Sort packages in ascending order
	sort.Strings(packages)

	return packages, nil
}

// getUbuntuReleaseName maps Ubuntu version to release name
func getUbuntuReleaseName(version string) string {
	switch {
	case strings.HasPrefix(version, "24.04"):
		return "noble"
	case strings.HasPrefix(version, "23.10"):
		return "mantic"
	case strings.HasPrefix(version, "22.04"):
		return "jammy"
	case strings.HasPrefix(version, "20.04"):
		return "focal"
	case strings.HasPrefix(version, "18.04"):
		return "bionic"
	case strings.HasPrefix(version, "16.04"):
		return "xenial"
	case strings.HasPrefix(version, "14.04"):
		return "trusty"
	default:
		return "unknown"
	}
}

// GetDefaultPackages fetches and returns the default package list for a given OS type and version
func GetDefaultPackages() ([]string, error) {
	h, err := host.Info()
	if err != nil {
		return nil, err
	}

	osType := h.Platform
	version := h.PlatformVersion

	baseURL := ""
	secondaryURL := ""
	fallbackURL := ""
	switch strings.ToLower(osType) {
	case "centos":
		baseURL = fmt.Sprintf("https://mirror.stream.centos.org/%s/BaseOS/x86_64/os/repodata/repomd.xml", version)
		secondaryURL = fmt.Sprintf("https://vault.centos.org/%s/BaseOS/x86_64/os/repodata/repomd.xml", version)
		fallbackURL = fmt.Sprintf("https://vault.centos.org/%s/os/x86_64/repodata/repomd.xml", version)
	case "redhat":
		fallthrough
	case "rocky":
		baseURL = fmt.Sprintf("https://dl.rockylinux.org/pub/rocky/%s/BaseOS/x86_64/os/repodata/repomd.xml", version)
		secondaryURL = fmt.Sprintf("https://dl.rockylinux.org/vault/rocky/%s/BaseOS/x86_64/os/repodata/repomd.xml", version)
	case "ubuntu":
		releaseName := getUbuntuReleaseName(version)
		if releaseName == "unknown" {
			errMsg := fmt.Sprintf("packageFilter: unsupported Ubuntu version: %s", version)
			logger.Println(logger.ERROR, true, errMsg)
			return nil, errors.New(errMsg)
		}
		baseURL = fmt.Sprintf("https://cloud-images.ubuntu.com/minimal/releases/%s/release/ubuntu-%s-minimal-cloudimg-amd64.manifest", releaseName, version)
	default:
		errMsg := fmt.Sprintf("packageFilter: unsupported OS Type: %s", osType)
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	if osType == "ubuntu" {
		// Fetch Ubuntu manifest file
		logger.Println(logger.INFO, false, "packageFilter: Fetching Ubuntu manifest from:", baseURL)
		manifestData, err := fetchURL(baseURL)
		if err != nil {
			errMsg := "packageFilter: error fetching Ubuntu manifest: " + err.Error()
			logger.Println(logger.ERROR, true, errMsg)
			return nil, errors.New(errMsg)
		}

		// Parse the manifest file to extract package names
		packages, err := parseUbuntuManifest(manifestData)
		if err != nil {
			errMsg := "packageFilter: error parsing Ubuntu manifest: " + err.Error()
			logger.Println(logger.ERROR, true, errMsg)
			return nil, errors.New(errMsg)
		}

		return packages, nil
	}

	// Fetch repomd.xml for CentOS, RockyLinux, or RedHat
	logger.Println(logger.INFO, false, "packageFilter: Fetching repomd.xml from:", baseURL)
	repomdData, err := fetchURL(baseURL)
	if err != nil && secondaryURL != "" && strings.Contains(err.Error(), "404") {
		// If there's a 404 error and we have a secondary URL, try fetching from the secondary URL
		logger.Println(logger.WARN, false, "packageFilter: Primary URL failed, trying secondary URL:", secondaryURL)
		baseURL = secondaryURL
		repomdData, err = fetchURL(baseURL)
		if err != nil && fallbackURL != "" && strings.Contains(err.Error(), "404") {
			// If there's a 404 error and we have a fallback URL, try fetching from the fallback URL
			logger.Println(logger.WARN, false, "packageFilter: Secondary URL failed, trying fallback URL:", fallbackURL)
			baseURL = fallbackURL
			repomdData, err = fetchURL(baseURL)
			if err != nil {
				errMsg := "packageFilter: error fetching repomd.xml from fallback URL: " + err.Error()
				logger.Println(logger.ERROR, true, errMsg)
				return nil, errors.New(errMsg)
			}
		}
	} else if err != nil {
		errMsg := "packageFilter: error fetching repomd.xml: " + err.Error()
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	// Parse repomd.xml to find the group_gz or group_xz location
	groupFileURL, err := parseRepoMD(repomdData)
	if err != nil {
		errMsg := "packageFilter: error parsing repomd.xml: " + err.Error()
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	groupFileFullURL := strings.TrimSuffix(baseURL, "repodata/repomd.xml") + groupFileURL

	// Fetch the group file and decompress it
	logger.Println(logger.INFO, false, "packageFilter: Fetching and decompressing group file from:", groupFileFullURL)
	groupData, err := fetchURL(groupFileFullURL)
	if err != nil {
		errMsg := "packageFilter: error fetching group file: " + err.Error()
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	// Parse the groups data to extract mandatory and default packages
	packages, err := parseGroups(groupData)
	if err != nil {
		errMsg := "packageFilter: error parsing group data: " + err.Error()
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	return packages, nil
}
