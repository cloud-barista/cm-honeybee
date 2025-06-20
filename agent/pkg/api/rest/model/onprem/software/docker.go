package software

import (
	"github.com/docker/docker/api/types/container"
)

type Docker struct {
	Containers []container.Summary
	//Images     []types.ImageMetadata
}
