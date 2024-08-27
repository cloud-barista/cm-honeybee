package privileged

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

var standardCapabilities = map[string]struct{}{
	"cap_chown":              {},
	"cap_dac_override":       {},
	"cap_dac_read_search":    {},
	"cap_fowner":             {},
	"cap_fsetid":             {},
	"cap_kill":               {},
	"cap_setgid":             {},
	"cap_setuid":             {},
	"cap_setpcap":            {},
	"cap_linux_immutable":    {},
	"cap_net_bind_service":   {},
	"cap_net_broadcast":      {},
	"cap_net_admin":          {},
	"cap_net_raw":            {},
	"cap_ipc_lock":           {},
	"cap_ipc_owner":          {},
	"cap_sys_module":         {},
	"cap_sys_rawio":          {},
	"cap_sys_chroot":         {},
	"cap_sys_ptrace":         {},
	"cap_sys_pacct":          {},
	"cap_sys_admin":          {},
	"cap_sys_boot":           {},
	"cap_sys_nice":           {},
	"cap_sys_resource":       {},
	"cap_sys_time":           {},
	"cap_sys_tty_config":     {},
	"cap_mknod":              {},
	"cap_lease":              {},
	"cap_audit_write":        {},
	"cap_audit_control":      {},
	"cap_setfcap":            {},
	"cap_mac_override":       {},
	"cap_mac_admin":          {},
	"cap_syslog":             {},
	"cap_wake_alarm":         {},
	"cap_block_suspend":      {},
	"cap_audit_read":         {},
	"cap_perfmon":            {},
	"cap_bpf":                {},
	"cap_checkpoint_restore": {},
}

func parseCapEff(capEff string) map[string]struct{} {
	caps := make(map[string]struct{})
	for i := 0; i < len(capEff); i += 2 {
		hexValue := capEff[i : i+2]
		if hexValue == "ff" {
			for _cap := range standardCapabilities {
				caps[_cap] = struct{}{}
			}
		}
	}
	return caps
}

func CheckPrivileged() error {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	var capEff string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "CapEff:") {
			capEff = strings.TrimSpace(line[7:])
			break
		}
	}

	if capEff == "" {
		return errors.New("no CapEff value in /proc/self/status")
	}

	activeCaps := parseCapEff(capEff)
	for _cap := range standardCapabilities {
		if _, ok := activeCaps[_cap]; !ok {
			return errors.New("capabilities not available (Please enable privileged mode if you running inside the container.)")
		}
	}

	return nil
}
