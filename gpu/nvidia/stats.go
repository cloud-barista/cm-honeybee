package nvidia

type AllGPUStats = map[string]GPUStats

type GPUStats struct {
	UsagePercentage    uint32 `json:"usagePercentage"`
	MemoryUsedInBytes  uint64 `json:"memoryUsedInBytes"`
	TotalMemoryInBytes uint64 `json:"totalMemoryInBytes"`
}
