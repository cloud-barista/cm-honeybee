package kubernetes

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"

	"helm.sh/helm/v3/pkg/release"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"

	"github.com/jollaman999/utils/logger"
)

type Release struct {
	Namespace string `json:"namespace"`
}

func GetHelmNamespaces() ([]string, error) {

	if !KubeConfigCheck() {
		return nil, fmt.Errorf(KubeConfigPath + ": no such file or directory")
	}

	cmd := exec.Command("helm", "ls", "-A", "-o", "json", "--kube-insecure-skip-tls-verify", "--kubeconfig", KubeConfigPath)

	output, err := cmd.Output()
	if err != nil {
		logger.Println(logger.ERROR, true, "Error executing helm command: "+err.Error())
		return []string{}, err
	}

	var releases []Release
	err = json.Unmarshal(output, &releases)
	if err != nil {
		logger.Println(logger.ERROR, true, "Error unmarshaling helm Namespace: "+err.Error())
		return []string{}, err
	}

	namespaceSet := make(map[string]struct{})
	for _, release := range releases {
		namespaceSet[release.Namespace] = struct{}{}
	}

	var namespaces []string
	for ns := range namespaceSet {
		namespaces = append(namespaces, ns)
	}

	sort.Strings(namespaces)

	return namespaces, nil
}

func GetRepoInfo() ([]kubernetes.Repo, error) {

	// $USER/.config/helm/repositories.yaml
	cmd := exec.Command("helm", "repo", "list", "-o", "json", "--kube-insecure-skip-tls-verify", "--kubeconfig", KubeConfigPath)

	output, err := cmd.Output()
	if err != nil {
		logger.Println(logger.ERROR, true, "Error executing helm command: "+err.Error())
		return []kubernetes.Repo{}, err
	}

	var repos []kubernetes.Repo
	err = json.Unmarshal(output, &repos)
	if err != nil {
		logger.Println(logger.ERROR, true, "Error unmarshaling helm Repo: "+err.Error())
		return []kubernetes.Repo{}, err
	}

	return repos, nil
}

func GetReleaseInfo() ([]kubernetes.Release, error) {

	namespaces, err := GetHelmNamespaces()
	if err != nil {
		return []kubernetes.Release{}, err
	}

	var releases []kubernetes.Release

	for _, ns := range namespaces {
		helmClientset, err := GetHelmClientSet(ns)
		if err != nil {
			logger.Println(logger.ERROR, true, "Helm Connection Error: "+err.Error())
			return []kubernetes.Release{}, err
		}

		var objects []*release.Release
		objects, err = helmClientset.ListDeployedReleases()
		if err != nil {
			return []kubernetes.Release{}, err
		}

		ObjectCnt := len(objects)

		for i := 0; i < ObjectCnt; i++ {

			object, err := helmClientset.GetRelease(objects[i].Name)
			if err != nil {
				return []kubernetes.Release{}, err
			}

			release := kubernetes.Release{
				Name:             object.Name,
				Namespace:        object.Namespace,
				Revision:         object.Version,
				Updated:          object.Info.LastDeployed.Time,
				Status:           string(object.Info.Status),
				AppVersion:       object.Chart.Metadata.AppVersion,
				ChartNameVersion: object.Chart.Metadata.Name + "-" + object.Chart.Metadata.Version,
			}
			releases = append(releases, release)
		}

	}

	return releases, nil
}
