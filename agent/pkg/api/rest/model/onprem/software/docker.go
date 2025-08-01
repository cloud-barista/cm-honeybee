package software

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

type Docker struct {
	ContainerSummary container.Summary
	ContainerInspect container.InspectResponse
	ImageInspect     image.InspectResponse
}
