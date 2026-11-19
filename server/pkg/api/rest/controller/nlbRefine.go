package controller

import (
	"strconv"
	"strings"

	inframodel "github.com/cloud-barista/cm-beetle/imdl/on-premise-model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
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
