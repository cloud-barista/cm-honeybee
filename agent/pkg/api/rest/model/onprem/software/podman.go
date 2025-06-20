package software

import (
	"github.com/docker/docker/api/types/container"
)

type Podman struct {
	Containers []container.Summary
	//Images     []types.ImageMetadata
}
