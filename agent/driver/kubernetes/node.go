package kubernetes

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"

	"github.com/jollaman999/utils/logger"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func convertToMiB(q resource.Quantity) int {
	bytes := q.Value()
	return int(bytes / 1048576)
}

func parseNodeSpec(nodeData map[string]interface{}, index int) kubernetes.NodeSpec {
	var nodeSpec kubernetes.NodeSpec

	if cpuStr, ok := common.GoJq(nodeData, fmt.Sprintf(".items[%d].status.capacity.cpu", index)).(string); ok {
		if cpu, err := strconv.Atoi(cpuStr); err == nil {
			nodeSpec.CPU = cpu
		}
	}

	if memStr, ok := common.GoJq(nodeData, fmt.Sprintf(".items[%d].status.capacity.memory", index)).(string); ok {
		if memQuantity, err := resource.ParseQuantity(memStr); err == nil {
			nodeSpec.Memory = convertToMiB(memQuantity)
		}
	}

	if storageStr, ok := common.GoJq(nodeData, fmt.Sprintf(".items[%d].status.capacity[\"ephemeral-storage\"]", index)).(string); ok {
		if storageQuantity, err := resource.ParseQuantity(storageStr); err == nil {
			nodeSpec.EphemeralStorage = convertToMiB(storageQuantity)
		}
	}

	return nodeSpec
}

func GetNodeInfo() (kubernetes.NodeCount, []kubernetes.Node, error) {
	clientset, err := GetKubernetesClientSet()
	if err != nil {
		logger.Println(logger.ERROR, true, "Kubernetes Connection Error: "+err.Error())
		return kubernetes.NodeCount{}, []kubernetes.Node{}, err
	}

	objects, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Println(logger.ERROR, true, "Nodes: "+err.Error())
		return kubernetes.NodeCount{}, []kubernetes.Node{}, err
	}

	nodeMap, err := common.Unmarshal(objects)
	if err != nil {
		logger.Println(logger.ERROR, true, "Error unmarshaling nodes: "+err.Error())
		return kubernetes.NodeCount{}, []kubernetes.Node{}, err
	}

	ObjectCnt := len(objects.Items)

	var nodeCount kubernetes.NodeCount
	var nodes []kubernetes.Node

	for i := 0; i < ObjectCnt; i++ {
		node := kubernetes.Node{
			Name:      common.GoJq(nodeMap, fmt.Sprintf(".items[%d].metadata.name", i)),
			Labels:    common.GoJq(nodeMap, fmt.Sprintf(".items[%d].metadata.labels", i)),
			Addresses: common.GoJq(nodeMap, fmt.Sprintf(".items[%d].status.addresses[]", i)),
			NodeSpec:  parseNodeSpec(nodeMap, i),
			NodeInfo:  common.GoJq(nodeMap, fmt.Sprintf(".items[%d].status.nodeInfo", i)),
		}

		nodeCount.Total++

		if _, ok := node.Labels.(map[string]interface{})["node-role.kubernetes.io/control-plane"]; ok {
			node.Type = kubernetes.NodeTypeControlPlane
			nodeCount.ControlPlane++
		} else if _, ok := node.Labels.(map[string]interface{})["node-role.kubernetes.io/master"]; ok {
			node.Type = kubernetes.NodeTypeControlPlane
			nodeCount.ControlPlane++
		} else {
			node.Type = kubernetes.NodeTypeWorker
			nodeCount.Worker++
		}

		nodes = append(nodes, node)
	}

	return nodeCount, nodes, nil
}
