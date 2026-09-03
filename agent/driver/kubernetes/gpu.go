package kubernetes

import (
	"sort"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"

	corev1 "k8s.io/api/core/v1"
)

// GPU vendors advertise their devices as extended resources under their own
// domain, and their feature discovery components label the node with the
// device details. Both are read here, because the resource name alone only
// says how many devices there are, not what they are.
const (
	nvidiaDomain = "nvidia.com/"
	amdDomain    = "amd.com/"
	intelDomain  = "gpu.intel.com/"
)

// parseNodeGPU lists the GPU extended resources a node advertises.
//
// Kubernetes has no core GPU resource, so a cluster that has GPUs states so
// only through these vendor-defined names. A node with none yields nil.
func parseNodeGPU(node corev1.Node) []kubernetes.NodeGPU {
	var gpus []kubernetes.NodeGPU

	for name, capacity := range node.Status.Capacity {
		resourceName := string(name)
		if !isGPUResource(resourceName) {
			continue
		}

		gpu := kubernetes.NodeGPU{
			ResourceName: resourceName,
			Capacity:     capacity.Value(),
			Vendor:       gpuVendor(resourceName),
		}

		if allocatable, ok := node.Status.Allocatable[name]; ok {
			gpu.Allocatable = allocatable.Value()
		}

		describeGPU(&gpu, node.Labels)

		gpus = append(gpus, gpu)
	}

	// Map iteration order is random, so sort by resource name to keep the
	// reported order stable between collections.
	sort.Slice(gpus, func(i, j int) bool { return gpus[i].ResourceName < gpus[j].ResourceName })

	return gpus
}

// isGPUResource reports whether an extended resource name names a GPU. The
// name must be vendor-qualified, which rules out the core resources, and MIG
// instances are matched explicitly because their name says "mig" rather than
// "gpu".
func isGPUResource(name string) bool {
	if !strings.Contains(name, "/") {
		return false
	}

	lower := strings.ToLower(name)

	return strings.Contains(lower, "gpu") || strings.HasPrefix(lower, nvidiaDomain+"mig-")
}

func gpuVendor(resourceName string) string {
	switch {
	case strings.HasPrefix(resourceName, nvidiaDomain):
		return "nvidia"
	case strings.HasPrefix(resourceName, amdDomain):
		return "amd"
	case strings.HasPrefix(resourceName, intelDomain):
		return "intel"
	}

	return strings.SplitN(resourceName, "/", 2)[0]
}

// describeGPU fills in the device details from the node labels written by the
// vendor's feature discovery. A cluster running only the bare device plugin
// carries none of these labels, so the fields stay empty rather than guessed.
func describeGPU(gpu *kubernetes.NodeGPU, labels map[string]string) {
	switch gpu.Vendor {
	case "nvidia":
		gpu.Product = firstLabel(labels, "nvidia.com/gpu.product")
		gpu.DriverVersion = firstLabel(labels,
			"nvidia.com/cuda.driver-version.full", "nvidia.com/cuda.driver.full")
		if gpu.DriverVersion == "" {
			gpu.DriverVersion = joinVersion(labels,
				"nvidia.com/cuda.driver.major", "nvidia.com/cuda.driver.minor", "nvidia.com/cuda.driver.rev")
		}
		gpu.Memory = intLabel(labels, "nvidia.com/gpu.memory")
		gpu.MIGCapable = firstLabel(labels, "nvidia.com/mig.capable")
		gpu.MIGStrategy = firstLabel(labels, "nvidia.com/mig.strategy")
	case "amd":
		gpu.Product = firstLabel(labels,
			"amd.com/gpu.product-name", "amd.com/gpu.device-id", "amd.com/gpu.family")
		gpu.DriverVersion = firstLabel(labels, "amd.com/gpu.driver-version")
		gpu.Memory = intLabel(labels, "amd.com/gpu.vram")
	case "intel":
		gpu.Product = firstLabel(labels, "gpu.intel.com/device-id", "gpu.intel.com/family")
	}
}

func firstLabel(labels map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(labels[key]); value != "" {
			return value
		}
	}

	return ""
}

func intLabel(labels map[string]string, key string) int {
	value, err := strconv.Atoi(strings.TrimSpace(labels[key]))
	if err != nil {
		return 0
	}

	return value
}

// joinVersion rebuilds a dotted version from the split labels older feature
// discovery releases wrote.
func joinVersion(labels map[string]string, keys ...string) string {
	parts := make([]string, 0, len(keys))

	for _, key := range keys {
		value := strings.TrimSpace(labels[key])
		if value == "" {
			break
		}
		parts = append(parts, value)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ".")
}
