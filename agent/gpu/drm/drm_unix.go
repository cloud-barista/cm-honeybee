// Getting DRM information for Linux & Unix like systems

//go:build !windows

package drm

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/NeowayLabs/drm"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/jollaman999/utils/logger"
)

const (
	driPath   = "/dev/dri"
	sysDRMDir = "/sys/class/drm"
)

// GetDRMInfo reads the kernel DRM driver behind each /dev/dri/cardN.
//
// The cards are enumerated here rather than through drm.ListDevices so that
// the card number survives into the result and so that each device file is
// closed again; without the card number the entries cannot be lined up with
// the GPUs nvidia-smi or rocm-smi report.
func GetDRMInfo() ([]infra.DRM, error) {
	cards, err := listCards()
	if err != nil {
		errMsg := "drm: " + err.Error()
		logger.Println(logger.DEBUG, true, errMsg)

		return []infra.DRM{}, errors.New(errMsg)
	}

	if len(cards) == 0 {
		errMsg := "drm: DRM is not available"
		logger.Println(logger.DEBUG, true, errMsg)

		return []infra.DRM{}, errors.New(errMsg)
	}

	d := make([]infra.DRM, 0, len(cards))

	for _, card := range cards {
		file, err := drm.OpenCard(card)
		if err != nil {
			// Opening a card needs read-write access to the device file, so a
			// card can legitimately be unreadable for the agent's user.
			logger.Println(logger.DEBUG, false,
				"drm: cannot open card"+strconv.Itoa(card)+": "+err.Error())

			continue
		}

		version, err := drm.GetVersion(file)
		_ = file.Close()

		if err != nil {
			logger.Println(logger.DEBUG, false,
				"drm: cannot read card"+strconv.Itoa(card)+" version: "+err.Error())

			continue
		}

		name := "card" + strconv.Itoa(card)

		d = append(d, infra.DRM{
			Card:       name,
			PCIBusID:   cardPCIBusID(name),
			DriverName: version.Name,
			DriverVersion: strconv.Itoa(int(version.Major)) + "." +
				strconv.Itoa(int(version.Minor)) + "." + strconv.Itoa(int(version.Patch)),
			DriverDate:        version.Date,
			DriverDescription: version.Desc,
		})
	}

	if len(d) == 0 {
		errMsg := "drm: no DRM card could be read"
		logger.Println(logger.DEBUG, true, errMsg)

		return []infra.DRM{}, errors.New(errMsg)
	}

	return d, nil
}

// listCards returns the card numbers present under /dev/dri, in ascending
// order.
func listCards() ([]int, error) {
	entries, err := os.ReadDir(driPath)
	if err != nil {
		return nil, err
	}

	var cards []int

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "card") {
			continue
		}

		card, err := strconv.Atoi(strings.TrimPrefix(name, "card"))
		if err != nil {
			continue
		}

		cards = append(cards, card)
	}

	sort.Ints(cards)

	return cards, nil
}

// cardPCIBusID resolves the PCI address a DRM card sits on, formatted the way
// nvidia-smi prints it (eight-digit domain, upper case) so the two can be
// compared directly. It returns an empty string for a card that is not on a
// PCI bus or whose sysfs entry is unreadable.
func cardPCIBusID(name string) string {
	target, err := filepath.EvalSymlinks(filepath.Join(sysDRMDir, name, "device"))
	if err != nil {
		return ""
	}

	address := filepath.Base(target)

	// A PCI address is domain:bus:device.function, for example 0000:01:00.0.
	parts := strings.Split(address, ":")
	if len(parts) != 3 {
		return ""
	}

	domain := parts[0]
	for len(domain) < 8 {
		domain = "0" + domain
	}

	return strings.ToUpper(domain + ":" + parts[1] + ":" + parts[2])
}
