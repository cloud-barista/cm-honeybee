package kubernetes

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"

	"github.com/jollaman999/utils/logger"
)

type Release struct {
	Namespace string `json:"namespace"`
}

func GetHelmNamespaces() ([]string, error) {

	cfg, err := GetHelmConfig("")
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to GetHelmConfig: "+err.Error())
		return nil, err
	}

	list := action.NewList(cfg)
	list.All = true
	list.AllNamespaces = true

	releases, err := list.Run()
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to list helm releases: "+err.Error())
		return nil, err
	}

	nsMap := make(map[string]struct{})
	for _, rel := range releases {
		nsMap[rel.Namespace] = struct{}{}
	}

	var namespaces []string
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	return namespaces, nil
}

func GetRepoInfo() ([]kubernetes.Repo, error) {

	repoFile := settings.RepositoryConfig // ~/.config/helm/repositories.yaml

	yamlBytes, err := os.ReadFile(repoFile)
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to read helm repo config: "+err.Error())
		return nil, err
	}

	var rf repo.File
	if err := yaml.Unmarshal(yamlBytes, &rf); err != nil {
		logger.Println(logger.ERROR, true, "Failed to parse repo config: "+err.Error())
		return nil, err
	}

	var repos []kubernetes.Repo
	for _, entry := range rf.Repositories {
		repos = append(repos, kubernetes.Repo{
			Name: entry.Name,
			URL:  entry.URL,
		})
	}
	return repos, nil
}

func GetReleaseInfo() ([]kubernetes.Release, error) {
	namespaces, err := GetHelmNamespaces()
	if err != nil {
		return nil, err
	}

	var releases []kubernetes.Release

	for _, ns := range namespaces {
		cfg, err := GetHelmConfig(ns)
		if err != nil {
			logger.Println(logger.ERROR, true, fmt.Sprintf("Helm config init failed for %s: %s", ns, err.Error()))
			continue
		}

		list := action.NewList(cfg)
		list.All = false
		list.Deployed = true

		releaseList, err := list.Run()
		if err != nil {
			logger.Println(logger.ERROR, true, fmt.Sprintf("Failed to list releases in %s: %s", ns, err.Error()))
			continue
		}

		for _, release := range releaseList {
			releases = append(releases, kubernetes.Release{
				Name:             release.Name,
				Namespace:        release.Namespace,
				Revision:         release.Version,
				Updated:          release.Info.LastDeployed.Time,
				Status:           string(release.Info.Status),
				AppVersion:       release.Chart.AppVersion(),
				ChartNameVersion: fmt.Sprintf("%s-%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version),
			})
		}
	}
	return releases, nil
}
