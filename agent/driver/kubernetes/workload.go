package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"

	"github.com/jollaman999/utils/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var kindList = []string{
	"namespaces",
	"pods",
	"services",
	"deployments",
	"daemonsets",
	"replicasets",
	"statefulsets",
	"job",
	"cronjobs",
	"ingresses",
	"persistentvolumes",
	"persistentvolumeclaims",
	"storageclasses",
	"configmaps",
	"servicesaccounts",
	"secrets",
	"roles",
	"rolebindings",
	"clusterroles",
	"clusterrolebindings",
}

var clientFilter = map[string]string{
	"namespaces":             "CoreV1.Namespaces",
	"pods":                   "CoreV1.Pods.\"\"",
	"services":               "CoreV1.Services.\"\"",
	"configmaps":             "CoreV1.ConfigMaps.\"\"",
	"servicesaccounts":       "CoreV1.ServiceAccounts.\"\"",
	"secrets":                "CoreV1.Secrets.\"\"",
	"persistentvolumes":      "CoreV1.PersistentVolumes",
	"persistentvolumeclaims": "CoreV1.PersistentVolumeClaims.\"\"",
	"deployments":            "AppsV1.Deployments.\"\"",
	"daemonsets":             "AppsV1.DaemonSets.\"\"",
	"replicasets":            "AppsV1.ReplicaSets.\"\"",
	"statefulsets":           "AppsV1.StatefulSets.\"\"",
	"job":                    "BatchV1.Jobs.\"\"",
	"cronjobs":               "BatchV1.CronJobs.\"\"",
	"ingresses":              "NetworkingV1.Ingresses.\"\"",
	"storageclasses":         "StorageV1.StorageClasses",
}

func GetWorkloadInfo() (map[string]interface{}, error) {
	workloads := make(map[string]interface{})

	for _, kind := range kindList {
		if clientMethod, exists := clientFilter[kind]; exists {
			objects, err := callClientMethod(clientMethod)
			if err != nil {
				logger.Println(logger.ERROR, true, fmt.Sprintf("Error fetching %s: %s", kind, err.Error()))
				continue
			}

			processObjects(kind, objects, workloads)
		}
	}

	return workloads, nil
}

func callClientMethod(methodName string) (interface{}, error) {
	clientset, err := GetKubernetesClientSet()
	if err != nil {
		logger.Println(logger.ERROR, true, "Kubernetes Connection Error: "+err.Error())
		return nil, err
	}

	methods := strings.Split(methodName, ".")
	var result = reflect.ValueOf(clientset)

	for _, method := range methods {
		if method == "\"\"" {
			result = result.Call([]reflect.Value{reflect.ValueOf("")})[0]
		} else {
			result = result.MethodByName(method)
			if !result.IsValid() {
				return nil, fmt.Errorf("invalid method: %s in %s", method, methodName)
			}
			if result.Type().NumIn() == 0 {
				result = result.Call(nil)[0]
			}
		}
	}

	listMethod := result.MethodByName("List")
	if !listMethod.IsValid() {
		return nil, fmt.Errorf("invalid List method for: %s", methodName)
	}

	results := listMethod.Call([]reflect.Value{
		reflect.ValueOf(context.TODO()),
		reflect.ValueOf(metav1.ListOptions{}),
	})

	if len(results) != 2 {
		return nil, fmt.Errorf("unexpected number of return values")
	}

	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	return results[0].Interface(), nil
}

func processObjects(kind string, objects interface{}, workloads map[string]interface{}) {

	objectMap, err := common.Unmarshal(objects)
	if err != nil {
		logger.Println(logger.ERROR, true, fmt.Sprintf("Error unmarshaling %s: %s", kind, err.Error()))
		return
	}

	items := common.GoJq(objectMap, ".items")
	objectCnt := 0
	if itemsSlice, ok := items.([]interface{}); ok {
		objectCnt = len(itemsSlice)
	}

	var newStruct []map[string]interface{}

	for i := 0; i < objectCnt; i++ {
		item := make(map[string]interface{})

		if namespace := common.GoJq(objectMap, fmt.Sprintf(".items[%d].metadata.namespace", i)); namespace != nil {
			item["Namespace"] = namespace
		}
		if name := common.GoJq(objectMap, fmt.Sprintf(".items[%d].metadata.name", i)); name != nil {
			item["Name"] = name
		}
		if svcType := common.GoJq(objectMap, fmt.Sprintf(".items[%d].spec.type", i)); svcType != nil {
			item["Type"] = svcType
		}
		if svcClusterIP := common.GoJq(objectMap, fmt.Sprintf(".items[%d].spec.clusterIP", i)); svcClusterIP != nil {
			item["ClusterIP"] = svcClusterIP
		}
		if status := common.GoJq(objectMap, fmt.Sprintf(".items[%d].status.phase", i)); status != nil {
			item["Status"] = status
		}

		if len(item) > 0 {
			newStruct = append(newStruct, item)
		}
	}

	workloads[kind] = newStruct
}
