package nvidia

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jollaman999/utils/logger"
)

// Result is the outcome of one nvidia-smi query.
type Result struct {
	GPUs []infra.NVIDIA
	// Schema is the nvidia-smi XML schema version the output was read with.
	Schema string
}

// QueryGPU collects every NVIDIA GPU visible to nvidia-smi.
//
// It returns an empty result and an error when nvidia-smi is missing, when the
// driver does not answer, or when the output does not parse. A host with no
// NVIDIA GPU is not an error: nvidia-smi answers with an empty device list.
func QueryGPU() (Result, error) {
	output, err := runNVIDIASmi(queryTimeout, "-q", "-x")
	if err != nil {
		logger.Println(logger.DEBUG, false, err.Error())

		return Result{GPUs: []infra.NVIDIA{}}, err
	}

	gpus, schema, err := parse(output)
	if err != nil {
		logger.Println(logger.DEBUG, false, "NVIDIA: failed to parse nvidia-smi output: "+err.Error())

		return Result{GPUs: []infra.NVIDIA{}, Schema: schema}, err
	}

	// The NVML version is a host-wide value that the query output does not
	// carry, so it is read separately and stamped onto every GPU.
	if version := getNVMLVersion(); version != "" {
		for i := range gpus {
			gpus[i].DeviceAttribute.NVMLVersion = version
		}
	}

	if gpus == nil {
		gpus = []infra.NVIDIA{}
	}

	return Result{GPUs: gpus, Schema: schema}, nil
}
