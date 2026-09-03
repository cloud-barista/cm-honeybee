package kubernetes

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func node(labels map[string]string, capacity, allocatable map[string]string) corev1.Node {
	toList := func(m map[string]string) corev1.ResourceList {
		list := corev1.ResourceList{}
		for name, value := range m {
			list[corev1.ResourceName(name)] = resource.MustParse(value)
		}

		return list
	}

	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Labels: labels},
		Status: corev1.NodeStatus{
			Capacity:    toList(capacity),
			Allocatable: toList(allocatable),
		},
	}
}

func TestParseNodeGPUNVIDIA(t *testing.T) {
	gpus := parseNodeGPU(node(
		map[string]string{
			"nvidia.com/gpu.product":              "NVIDIA-A100-SXM4-40GB",
			"nvidia.com/gpu.memory":               "40960",
			"nvidia.com/gpu.count":                "4",
			"nvidia.com/cuda.driver-version.full": "535.104.05",
			"nvidia.com/mig.capable":              "true",
			"nvidia.com/mig.strategy":             "single",
		},
		map[string]string{"cpu": "64", "memory": "512Gi", "nvidia.com/gpu": "4"},
		map[string]string{"cpu": "64", "memory": "512Gi", "nvidia.com/gpu": "3"},
	))

	if len(gpus) != 1 {
		t.Fatalf("got %d GPU resources, want 1", len(gpus))
	}

	g := gpus[0]

	if g.ResourceName != "nvidia.com/gpu" || g.Vendor != "nvidia" {
		t.Errorf("resource = %q, vendor = %q", g.ResourceName, g.Vendor)
	}

	if g.Capacity != 4 || g.Allocatable != 3 {
		t.Errorf("capacity/allocatable = %d/%d, want 4/3", g.Capacity, g.Allocatable)
	}

	if g.Product != "NVIDIA-A100-SXM4-40GB" {
		t.Errorf("product = %q", g.Product)
	}

	if g.DriverVersion != "535.104.05" {
		t.Errorf("driver_version = %q", g.DriverVersion)
	}

	if g.Memory != 40960 {
		t.Errorf("memory = %d, want 40960", g.Memory)
	}

	if g.MIGCapable != "true" || g.MIGStrategy != "single" {
		t.Errorf("mig = %q/%q, want true/single", g.MIGCapable, g.MIGStrategy)
	}
}

// Older feature discovery releases split the driver version across three
// labels instead of writing the full one.
func TestParseNodeGPUSplitDriverVersion(t *testing.T) {
	gpus := parseNodeGPU(node(
		map[string]string{
			"nvidia.com/cuda.driver.major": "525",
			"nvidia.com/cuda.driver.minor": "125",
			"nvidia.com/cuda.driver.rev":   "06",
		},
		map[string]string{"nvidia.com/gpu": "1"},
		map[string]string{"nvidia.com/gpu": "1"},
	))

	if len(gpus) != 1 {
		t.Fatalf("got %d GPU resources, want 1", len(gpus))
	}

	if gpus[0].DriverVersion != "525.125.06" {
		t.Errorf("driver_version = %q, want 525.125.06", gpus[0].DriverVersion)
	}
}

func TestParseNodeGPUAMDAndMIG(t *testing.T) {
	gpus := parseNodeGPU(node(
		map[string]string{
			"amd.com/gpu.product-name":   "Instinct-MI250X",
			"amd.com/gpu.vram":           "65536",
			"amd.com/gpu.driver-version": "6.7.0",
		},
		map[string]string{
			"cpu":                   "128",
			"amd.com/gpu":           "8",
			"nvidia.com/mig-1g.5gb": "7",
		},
		map[string]string{
			"cpu":                   "128",
			"amd.com/gpu":           "8",
			"nvidia.com/mig-1g.5gb": "7",
		},
	))

	if len(gpus) != 2 {
		t.Fatalf("got %d GPU resources, want 2", len(gpus))
	}

	// Sorted by resource name, so amd.com/gpu comes first.
	amd, mig := gpus[0], gpus[1]

	if amd.ResourceName != "amd.com/gpu" || amd.Vendor != "amd" {
		t.Errorf("first = %q/%q", amd.ResourceName, amd.Vendor)
	}

	if amd.Product != "Instinct-MI250X" || amd.Memory != 65536 || amd.DriverVersion != "6.7.0" {
		t.Errorf("amd details = %+v", amd)
	}

	// A MIG instance is advertised per profile and its name says mig, not gpu.
	if mig.ResourceName != "nvidia.com/mig-1g.5gb" || mig.Vendor != "nvidia" {
		t.Errorf("second = %q/%q", mig.ResourceName, mig.Vendor)
	}

	if mig.Capacity != 7 {
		t.Errorf("mig capacity = %d, want 7", mig.Capacity)
	}
}

// A node without any GPU must report none rather than an empty entry, and the
// core resources must never be mistaken for GPUs.
func TestParseNodeGPUNone(t *testing.T) {
	gpus := parseNodeGPU(node(
		map[string]string{"kubernetes.io/arch": "amd64"},
		map[string]string{"cpu": "8", "memory": "32Gi", "ephemeral-storage": "100Gi", "pods": "110"},
		map[string]string{"cpu": "8", "memory": "32Gi"},
	))

	if len(gpus) != 0 {
		t.Fatalf("got %+v, want no GPU resources", gpus)
	}
}

// The device plugin can run without feature discovery, and then the count is
// all the cluster knows. Nothing may be invented to fill the gap.
func TestParseNodeGPUWithoutFeatureDiscovery(t *testing.T) {
	gpus := parseNodeGPU(node(
		map[string]string{},
		map[string]string{"nvidia.com/gpu": "2"},
		map[string]string{"nvidia.com/gpu": "2"},
	))

	if len(gpus) != 1 {
		t.Fatalf("got %d GPU resources, want 1", len(gpus))
	}

	g := gpus[0]

	if g.Capacity != 2 {
		t.Errorf("capacity = %d, want 2", g.Capacity)
	}

	if g.Product != "" || g.DriverVersion != "" || g.Memory != 0 {
		t.Errorf("details = %+v, want all empty", g)
	}
}

func TestIsGPUResource(t *testing.T) {
	cases := map[string]bool{
		"nvidia.com/gpu":        true,
		"amd.com/gpu":           true,
		"gpu.intel.com/i915":    true,
		"nvidia.com/mig-1g.5gb": true,
		"cpu":                   false,
		"memory":                false,
		"ephemeral-storage":     false,
		"pods":                  false,
		"hugepages-2Mi":         false,
	}

	for name, want := range cases {
		if got := isGPUResource(name); got != want {
			t.Errorf("isGPUResource(%q) = %v, want %v", name, got, want)
		}
	}
}
