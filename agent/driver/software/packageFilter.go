package software

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/cavaliergopher/rpm"
	"github.com/jollaman999/utils/logger"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/ulikunitz/xz"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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
		Timeout:   time.Second * 30,
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
		if strings.ToLower(group.ID) == "core" || strings.ToLower(group.Name) == "core" ||
			strings.ToLower(group.ID) == "base" || strings.ToLower(group.Name) == "base" {
			for _, pkg := range group.PackageList {
				if pkg.Type == "mandatory" || pkg.Type == "default" {
					packages = append(packages, pkg.Name)
				}
			}
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

// fetchAndParseRPMDependencies fetches the RPM file and parses its dependencies
func fetchAndParseRPMDependencies(rpmURL string) ([]string, error) {
	logger.Println(logger.INFO, false, "packageFilter: Fetching RPM from:", rpmURL)
	rpmData, err := fetchURL(rpmURL)
	if err != nil {
		errMsg := "packageFilter: error fetching RPM: " + err.Error()
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	// Parse the RPM file to extract dependencies
	rpmReader := bytes.NewReader(rpmData)
	pkg, err := rpm.Read(rpmReader)
	if err != nil {
		errMsg := "packageFilter: error reading RPM file: " + err.Error()
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	var dependencies []string
	for _, req := range pkg.Requires() {
		dependencies = append(dependencies, req.Name())
	}

	// Sort dependencies in ascending order
	sort.Strings(dependencies)

	return dependencies, nil
}

// fetchDirectoryListing fetches and returns a list of RPM filenames from a directory URL.
func fetchDirectoryListing(dirURL string) ([]string, error) {
	logger.Println(logger.INFO, false, "Fetching directory listing from: ", dirURL)

	resp, err := http.Get(dirURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Failed to fetch directory listing from %s: %s", dirURL, resp.Status)
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the HTML body to find all links ending with ".rpm"
	var rpmFiles []string
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.Contains(line, ".rpm") {
			start := strings.Index(line, "href=\"")
			if start == -1 {
				continue
			}
			start += len("href=\"")
			end := strings.Index(line[start:], "\"")
			if end == -1 {
				continue
			}
			fileName := line[start : start+end]
			if strings.HasSuffix(fileName, ".rpm") {
				rpmFiles = append(rpmFiles, fileName)
			}
		}
	}

	return rpmFiles, nil
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

	var packages []string

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
		packages, err = parseUbuntuManifest(manifestData)
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
	packages, err = parseGroups(groupData)
	if err != nil {
		errMsg := "packageFilter: error parsing group data: " + err.Error()
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	// Determine if the version is below 9
	var isVersionBelowNine bool
	for i := 8; i >= 0; i-- {
		isVersionBelowNine = strings.HasPrefix(version, strconv.Itoa(i))
		if isVersionBelowNine {
			break
		}
	}

	// For rocky, it has first letter subdirectory starting from 8.5. redhat uses same information of rocky.
	if osType == "redhat" || osType == "rocky" {
		for i := 9; i >= 5; i-- {
			matched := strings.HasPrefix(version, "8."+strconv.Itoa(i))
			if matched {
				isVersionBelowNine = false
				break
			}
		}
	}

	// Fetch RPM versions for all default packages
	var allPackagesWithDependencies []string

	var routineMax = 50
	var wait sync.WaitGroup
	var mutex = &sync.Mutex{}
	var lenPackages = len(packages)

	// For each package, fetch and parse RPM dependencies
	if isVersionBelowNine {
		// For versions below 9, no first letter subdirectory
		dirURL := strings.ReplaceAll(baseURL, "repodata/repomd.xml", "Packages/")

		// Fetch RPM files in the directory and cache it
		rpmFiles, err := fetchDirectoryListing(dirURL)
		if err != nil {
			errMsg := "packageFilter: Error fetching directory listing: " + err.Error()
			logger.Println(logger.ERROR, true, errMsg)
			return nil, errors.New(errMsg)
		}

		for i := 0; i < lenPackages; {
			if lenPackages-i < routineMax {
				routineMax = lenPackages - i
			}

			wait.Add(routineMax)

			for j := 0; j < routineMax; j++ {
				go func(wait *sync.WaitGroup, pkgName string) {
					defer func() {
						wait.Done()
					}()

					// Find the correct RPM for the package
					var matchedRPM string
					for _, rpmFile := range rpmFiles {
						if strings.HasPrefix(rpmFile, pkgName+"-") {
							matchedRPM = rpmFile
							break
						}
					}

					if matchedRPM == "" {
						logger.Println(logger.WARN, false, "packageFilter: No RPM found for package:", pkgName)
						return
					}

					// For versions below 9, RPM is directly in Packages directory
					rpmURL := strings.ReplaceAll(baseURL, "repodata/repomd.xml", "Packages/") + matchedRPM

					// Fetch dependencies using the RPM URL
					dependencies, err := fetchAndParseRPMDependencies(rpmURL)
					if err != nil {
						logger.Println(logger.ERROR, true, "packageFilter: Error fetching dependencies for package:", pkgName, err)
						return
					}

					mutex.Lock()
					allPackagesWithDependencies = append(allPackagesWithDependencies, dependencies...)
					mutex.Unlock()
				}(&wait, packages[i])

				i++
				if i == lenPackages {
					break
				}
			}

			wait.Wait()
		}
	} else {
		// Map to store directory listings for each first letter
		dirListings := make(map[string][]string)

		for i := 0; i < lenPackages; {
			if lenPackages-i < routineMax {
				routineMax = lenPackages - i
			}

			wait.Add(routineMax)

			for j := 0; j < routineMax; j++ {
				go func(wait *sync.WaitGroup, pkgName string) {
					defer func() {
						wait.Done()
					}()

					// Determine the first letter of the package name
					firstLetter := strings.ToLower(string(pkgName[0]))

					// Check if directory listing is already fetched
					if _, exists := dirListings[firstLetter]; !exists {
						// For versions 9 and above, use first letter subdirectory
						dirURL := strings.ReplaceAll(baseURL, "repodata/repomd.xml", "Packages/") + firstLetter

						// Fetch RPM files in the directory and cache it
						rpmFiles, err := fetchDirectoryListing(dirURL)
						if err != nil {
							logger.Println(logger.ERROR, true, "packageFilter: Error fetching directory listing:", err)
							return
						}
						dirListings[firstLetter] = rpmFiles
					}

					// Get the cached directory listing
					rpmFiles := dirListings[firstLetter]

					// Find the correct RPM for the package
					var matchedRPM string
					for _, rpmFile := range rpmFiles {
						if strings.HasPrefix(rpmFile, pkgName+"-") {
							matchedRPM = rpmFile
							break
						}
					}

					if matchedRPM == "" {
						logger.Println(logger.WARN, false, "packageFilter: No RPM found for package:", pkgName)
						return
					}

					// For versions 9 and above, RPM is in a subdirectory by first letter
					rpmURL := strings.ReplaceAll(baseURL, "repodata/repomd.xml", "Packages/") + fmt.Sprintf("%s/%s", firstLetter, matchedRPM)

					// Fetch dependencies using the RPM URL
					dependencies, err := fetchAndParseRPMDependencies(rpmURL)
					if err != nil {
						logger.Println(logger.ERROR, true, "packageFilter: Error fetching dependencies for package:", pkgName, err)
						return
					}

					mutex.Lock()
					allPackagesWithDependencies = append(allPackagesWithDependencies, dependencies...)
					mutex.Unlock()
				}(&wait, packages[i])

				i++
				if i == lenPackages {
					break
				}
			}

			wait.Wait()
		}
	}

	// Remove duplicates and sort the package list
	packageSet := make(map[string]struct{})
	for _, pkg := range allPackagesWithDependencies {
		// Skip if contained slash
		slashIdx := strings.Index(pkg, "/")
		if slashIdx != -1 {
			continue
		}

		// Find and remove text within parentheses, including the parentheses themselves
		for {
			openIdx := strings.Index(pkg, "(")
			closeIdx := strings.Index(pkg, ")")
			if openIdx != -1 && closeIdx != -1 && closeIdx > openIdx {
				// Remove the part within parentheses, including the parentheses
				pkg = pkg[:openIdx] + pkg[closeIdx+1:]
			} else {
				break // Exit the loop if no more parentheses are found
			}
		}

		// Trim spaces after removing text within parentheses
		pkg = strings.TrimSpace(pkg)

		// Remove text starting from ".so"
		soIdx := strings.Index(pkg, ".so")
		if soIdx != -1 {
			pkg = pkg[:soIdx]
		}

		packageSet[pkg] = struct{}{}
	}

	var uniquePackages []string
	for pkg := range packageSet {
		uniquePackages = append(uniquePackages, pkg)
	}
	sort.Strings(uniquePackages)

	return uniquePackages, nil
}
