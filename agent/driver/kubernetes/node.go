package infra

import (
	"context"
	"fmt"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"

	"github.com/jollaman999/utils/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNodeInfo() ([]kubernetes.Node, error) {
	clientset, err := GetClientSet()
	if err != nil {
		logger.Println(logger.ERROR, true, "Kubernetes Connection Error: "+err.Error())
		return []kubernetes.Node{}, err
	}

	objects, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Println(logger.ERROR, true, "Nodes: "+err.Error())
		return []kubernetes.Node{}, err
	}

	nodeMap, err := common.Unmarshal(objects)
	if err != nil {
		logger.Println(logger.ERROR, true, "Error unmarshaling nodes: "+err.Error())
		return []kubernetes.Node{}, err
	}

	ObjectCnt := len(objects.Items)

	var nodes []kubernetes.Node

	for i := 0; i < ObjectCnt; i++ {
		node := kubernetes.Node{
			Name:      common.GoJq(nodeMap, fmt.Sprintf(".items[%d].metadata.name", i)),
			Labels:    common.GoJq(nodeMap, fmt.Sprintf(".items[%d].metadata.labels", i)),
			Addresses: common.GoJq(nodeMap, fmt.Sprintf(".items[%d].status.addresses[]", i)),
			NodeInfo:  common.GoJq(nodeMap, fmt.Sprintf(".items[%d].status.nodeInfo", i)),
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
