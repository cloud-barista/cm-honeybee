//go:build linux

package network

import (
	"context"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/network"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"github.com/taigrr/systemctl"
	"net/netip"
	"strings"
	"time"
)

func isSystemdResolvedActive() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	active, _ := systemctl.IsActive(ctx, "systemd-resolved", systemctl.Options{UserMode: false})
	return active
}

func getDNSFromSystemdResolved() ([]string, error) {
	output, err := cmd.RunCMD("resolvectl status | grep -i 'dns servers' | sed 's/.*dns servers: //Ig'")
	if err != nil {
		return nil, err
	}

	var nameservers []string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		split := strings.Split(line, " ")
		nameservers = append(nameservers, split...)
	}

	return nameservers, nil
}

func getDNSFromResolvedConf() ([]string, error) {
	var nameservers []string

	data, err := fileutil.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nameservers, err
	}

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "nameserver" {
			continue
		}
		for _, field := range fields[1:] {
			ip, err := netip.ParseAddr(field)
			if err != nil {
				continue
			}
			nameservers = append(nameservers, ip.String())
		}
	}

	return nameservers, nil
}

func GetDNS() (network.DNS, error) {
	var err error
	var nameservers []string

	if isSystemdResolvedActive() {
		nameservers, err = getDNSFromSystemdResolved()
		if err != nil {
			logger.Println(logger.WARN, false, "DNS: Failed to get nameservers while systemd-resolved is running.")
			logger.Println(logger.WARN, false, "DNS: Fallback to get nameservers from /etc/resolv.conf.")
			nameservers, err = getDNSFromResolvedConf()
		}
	} else {
		nameservers, err = getDNSFromResolvedConf()
	}

	return network.DNS{DNSServer: nameservers}, err
}
