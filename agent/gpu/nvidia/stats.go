package nvidia

import (
	"strconv"

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

	fillMIGUUIDs(gpus)

	if gpus == nil {
		gpus = []infra.NVIDIA{}
	}

	return Result{GPUs: gpus, Schema: schema}, nil
}

// fillMIGUUIDs supplies the MIG instance UUIDs that the query output left out.
//
// Some drivers omit <uuid> from <mig_device> even though the schema carries it,
// and `nvidia-smi -L` still reports it. The extra command only runs when there
// is something to fill, so a host with no MIG instances, or one whose output
// already names them, pays nothing. A UUID that came from the query output is
// never overwritten: that output is the primary source and this is a fallback.
func fillMIGUUIDs(gpus []infra.NVIDIA) {
	missing := false

	for i := range gpus {
		for j := range gpus[i].MIGDevices {
			if gpus[i].MIGDevices[j].UUID == "" {
				missing = true

				break
			}
		}
	}

	if !missing {
		return
	}

	uuids := migUUIDsByDevice()
	if len(uuids) == 0 {
		return
	}

	applyMIGUUIDs(gpus, uuids)
}

// applyMIGUUIDs writes the listed UUIDs onto the MIG instances that have none.
// Both sides are numbered by the driver, so the GPU index and the MIG device
// index line the two up.
func applyMIGUUIDs(gpus []infra.NVIDIA, uuids map[int]map[int]string) {
	for i := range gpus {
		byDevice := uuids[gpus[i].DeviceAttribute.Index]
		if byDevice == nil {
			continue
		}

		for j := range gpus[i].MIGDevices {
			mig := &gpus[i].MIGDevices[j]
			if mig.UUID != "" {
				continue
			}

			// The MIG index is a string in the model because the query output
			// carries it as one; `-L` numbers the same devices.
			index, err := strconv.Atoi(mig.Index)
			if err != nil {
				continue
			}

			if uuid := byDevice[index]; uuid != "" {
				mig.UUID = uuid
			}
		}
	}
}
