package controller

import (
	"strconv"
	"strings"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
)

// haproxyToNLBs converts the collected HAProxy configuration into NLB entries,
// one per frontend-backend pair and one per listen section.
func haproxyToNLBs(haproxy infra.HAProxy, machineID string) []inframodel.NlbProperty {
	var nlbs []inframodel.NlbProperty

	for _, fe := range haproxy.Frontends {
		be, ok := findHAProxyBackend(haproxy.Backends, fe.DefaultBackend)
		if !ok {
			continue
		}
		mode := firstNonEmpty(be.Options["mode"], fe.Options["mode"], haproxy.Defaults["mode"])
		balance := firstNonEmpty(be.Balance, haproxy.Defaults["balance"])
		if nlb, ok := buildHAProxyNLB(machineID, fe.Bind, be.Name, balance, mode, be.Servers); ok {
			nlbs = append(nlbs, nlb)
		}
	}

	for _, ls := range haproxy.Listens {
		mode := firstNonEmpty(ls.Options["mode"], haproxy.Defaults["mode"])
		balance := firstNonEmpty(ls.Balance, haproxy.Defaults["balance"])
		if nlb, ok := buildHAProxyNLB(machineID, ls.Bind, ls.Name, balance, mode, ls.Servers); ok {
			nlbs = append(nlbs, nlb)
		}
	}

	return nlbs
}

// findHAProxyBackend returns the backend named by default_backend, or the only
// backend when the frontend names none.
func findHAProxyBackend(backends []infra.HAProxyBackend, name string) (infra.HAProxyBackend, bool) {
	for _, be := range backends {
		if name != "" && be.Name == name {
			return be, true
		}
	}
	if name == "" && len(backends) == 1 {
		return backends[0], true
	}
	return infra.HAProxyBackend{}, false
}

func buildHAProxyNLB(machineID, bind, backendName, balance, mode string, servers []infra.HAProxyServer) (inframodel.NlbProperty, bool) {
	bindAddress, port, ok := parseHAProxyBind(bind)
	if !ok {
		return inframodel.NlbProperty{}, false
	}

	backendServers, healthCheck := convertHAProxyServers(servers, port)
	if len(backendServers) == 0 {
		return inframodel.NlbProperty{}, false
	}

	backendProtocol := "tcp"
	if strings.EqualFold(firstField(mode), "http") {
		backendProtocol = "http"
	}

	return inframodel.NlbProperty{
		HostMachineId: machineID,
		Software:      "haproxy",
		Listener: inframodel.NlbListenerProperty{
			BindAddress: bindAddress,
			Port:        port,
			Protocol:    "tcp",
		},
		Backend: inframodel.NlbBackendProperty{
			Name:     backendName,
			Balance:  firstNonEmpty(firstField(balance), "roundrobin"),
			Protocol: backendProtocol,
			Servers:  backendServers,
		},
		HealthCheck: inframodel.NlbHealthCheckProperty{Enabled: healthCheck},
	}, true
}

// parseHAProxyBind reads the first address of a bind line such as "*:80",
// ":443 ssl crt ..." or "10.0.0.1:8080,10.0.0.1:8081". Wildcard addresses
// are reported as "*".
func parseHAProxyBind(bind string) (string, int, bool) {
	addr := strings.Split(firstField(bind), ",")[0]
	host, port, ok := splitHostPort(addr)
	if !ok || port == 0 {
		return "", 0, false
	}

	switch host {
	case "", "*", "0.0.0.0", "::":
		host = "*"
	}

	return host, port, true
}

// convertHAProxyServers converts backend servers, falling back to the listener
// port for servers declared without one, as HAProxy does. It also reports
// whether any server has health checks enabled.
func convertHAProxyServers(servers []infra.HAProxyServer, listenerPort int) ([]inframodel.NlbServerProperty, bool) {
	var out []inframodel.NlbServerProperty
	healthCheck := false

	for _, s := range servers {
		ip, port, ok := splitHostPort(s.Address)
		if !ok || ip == "" {
			continue
		}
		if port == 0 {
			port = listenerPort
		}

		weight := 0
		options := strings.Fields(s.Options)
		for i, opt := range options {
			switch opt {
			case "check":
				healthCheck = true
			case "weight":
				if i+1 < len(options) {
					weight, _ = strconv.Atoi(options[i+1])
				}
			}
		}

		out = append(out, inframodel.NlbServerProperty{
			Name:   s.Name,
			IP:     ip,
			Port:   port,
			Weight: weight,
		})
	}

	return out, healthCheck
}

// splitHostPort splits "host:port", "[v6]:port" or a bare host. A missing port
// is returned as 0.
func splitHostPort(addr string) (string, int, bool) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", 0, false
	}

	if strings.HasPrefix(addr, "[") {
		end := strings.Index(addr, "]")
		if end < 0 {
			return "", 0, false
		}
		host := addr[1:end]
		rest := strings.TrimPrefix(addr[end+1:], ":")
		if rest == "" {
			return host, 0, true
		}
		port, err := strconv.Atoi(rest)
		if err != nil || port < 1 || port > 65535 {
			return "", 0, false
		}
		return host, port, true
	}

	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, 0, true
	}
	port, err := strconv.Atoi(addr[idx+1:])
	if err != nil || port < 1 || port > 65535 {
		return "", 0, false
	}
	return addr[:idx], port, true
}

func firstField(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// cspNLBSoftware names a CSP-managed NLB in the refined model's Software
// field, which an NLB found in HAProxy's configuration fills with "haproxy".
// The provider is part of it, e.g. "aws-nlb", since the managed product
// differs per CSP; "csp-nlb" is used when the provider is not known.
func cspNLBSoftware(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "csp"
	}
	return provider + "-nlb"
}

// cspNLBBindAddress gives the refined listener address of a CSP NLB. cm-beetle
// reads "*" as a PUBLIC NLB and a specific address as an INTERNAL one. An
// INTERNAL NLB is given its first listener IP, or its DNS name on a driver that
// reports no IP (an AWS internal NLB); an NLB whose type the driver does not
// tell is left empty.
func cspNLBBindAddress(rec *model.CSPNLBRecord) string {
	switch rec.Type {
	case nlbTypePublic:
		return "*"
	case nlbTypeInternal:
		for _, ip := range strings.Split(rec.Listener.IP, ",") {
			if ip = strings.TrimSpace(ip); ip != "" {
				return ip
			}
		}
		if rec.Listener.DNSName != "" {
			return rec.Listener.DNSName
		}
		logger.Println(logger.WARN, false, "CSP NLB '"+rec.Name+"' is INTERNAL but reports neither a listener IP nor a DNS name;"+
			" its refined bind address is left empty")
		return ""
	default:
		return ""
	}
}

// cspNLBToNlbProperty maps a stored CSP NLB into the refined model's NLB entry.
// The drivers report the backend VMs by their identifiers only, so a server's
// IP is taken from vmIPs, the private IPs of the group's CSP VMs keyed by their
// CSP ID and name (cspVMPrivateIPs). A VM not in it keeps an empty IP, which
// cm-beetle cannot match to a node. A CSP NLB runs on no host of the group, so
// HostMachineId stays empty, and its balancing algorithm is not reported.
func cspNLBToNlbProperty(rec *model.CSPNLBRecord, provider string, vmIPs map[string]string) inframodel.NlbProperty {
	port := atoiPort(rec.VMGroup.Port)
	servers := make([]inframodel.NlbServerProperty, 0, len(rec.VMGroup.VMs))
	for _, vm := range rec.VMGroup.VMs {
		ip := vmIPs[vm.SystemID]
		if ip == "" && vm.Name != "" {
			ip = vmIPs[vm.Name]
		}
		servers = append(servers, inframodel.NlbServerProperty{
			Name: firstNonEmpty(vm.Name, vm.SystemID),
			IP:   ip,
			Port: port,
		})
	}

	hc := rec.HealthChecker
	return inframodel.NlbProperty{
		Software: cspNLBSoftware(provider),
		Listener: inframodel.NlbListenerProperty{
			BindAddress: cspNLBBindAddress(rec),
			Port:        atoiPort(rec.Listener.Port),
			Protocol:    strings.ToLower(rec.Listener.Protocol),
		},
		Backend: inframodel.NlbBackendProperty{
			Name:     rec.Name,
			Protocol: strings.ToLower(rec.VMGroup.Protocol),
			Servers:  servers,
		},
		HealthCheck: inframodel.NlbHealthCheckProperty{
			Enabled:   hc.Protocol != "" || hc.Interval > 0,
			Protocol:  strings.ToLower(hc.Protocol),
			Port:      atoiPort(hc.Port),
			Interval:  hc.Interval,
			Timeout:   hc.Timeout,
			Threshold: hc.Threshold,
		},
	}
}

// atoiPort reads a port a driver reported as text. Anything that is not a port
// number, such as an empty value, becomes 0.
func atoiPort(s string) int {
	port, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || port < 1 || port > 65535 {
		return 0
	}
	return port
}
