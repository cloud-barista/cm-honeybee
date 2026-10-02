package software

import (
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
)

type Container struct {
	ContainerSummary container.Summary
	ContainerInspect container.InspectResponse
	ImageInspect     image.InspectResponse
}
