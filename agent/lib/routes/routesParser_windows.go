//go:build windows

package routes

import (
	"errors"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/logger"
	"net"
	"regexp"
	"strconv"
	"strings"
)

type RouteStruct struct {
	Interface   string
	Destination string
	Netmask     string
	NextHop     string
	Metric      int
	Family      string
}

func cidrToNetmask(prefixLen int) string {
	if prefixLen < 0 || prefixLen > 32 {
		return "0.0.0.0"
	}

	mask := net.CIDRMask(prefixLen, 32)
	netmask := net.IPv4(mask[0], mask[1], mask[2], mask[3])
	return netmask.String()
}

// PowerShell Get-NetRoute output format with interface names:
// ifIndex DestinationPrefix             NextHop         RouteMetric InterfaceAlias
// ------- -----------------             -------         ----------- --------------
// 12      0.0.0.0/0                     192.168.110.254 256         Ethernet
// 12      192.168.110.0/24              0.0.0.0         256         Ethernet
// 1       ::1/128                       ::              256         Interface1
//
// This function uses PowerShell Get-NetRoute cmdlet to get both IPv4 and IPv6 routing information
// in a structured format that's easier to parse than traditional route print command.
//
// For on-link routes:
// - IPv4: NextHop shows as "0.0.0.0"
// - IPv6: NextHop shows as "::"
//
// If multiple default gateways are present, then the one with the lowest metric is returned.
// For IPv4: default route is 0.0.0.0/0
// For IPv6: default route is ::/0
func GetRoutes(getOnlyDefaults bool) ([]RouteStruct, error) {
	output, err := cmd.RunPowerShell("[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; " +
		"Get-NetRoute | " +
		"Select-Object ifIndex,DestinationPrefix,NextHop,RouteMetric," +
		"@{Name='InterfaceAlias';" +
		"Expression={$adapter = Get-NetAdapter -InterfaceIndex $_.ifIndex -ErrorAction SilentlyContinue; " +
		"if($adapter){$adapter.InterfaceAlias | " +
		"Select-Object -First 1}" +
		"else{$ipif = Get-NetIPInterface -InterfaceIndex $_.ifIndex -ErrorAction SilentlyContinue; " +
		"if($ipif){($ipif.InterfaceAlias | Select-Object -First 1)}" +
		"else{'Interface'+$_.ifIndex}}}}" +
		" | Format-Table -AutoSize")
	if err != nil {
		errMsg := err.Error()
		logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
		return nil, errors.New(errMsg)
	}

	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}(/\d{1,2})?$`)
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4}(/\d{1,3})?$|^::(/\d{1,3})?$`)

	var routes []RouteStruct
	lines := strings.Split(output, "\n")

	headerFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.Contains(line, "DestinationPrefix") && strings.Contains(line, "InterfaceAlias") {
			headerFound = true
			continue
		}

		if strings.HasPrefix(line, "---") {
			continue
		}

		if !headerFound {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		destinationPrefix := fields[1]
		nextHop := fields[2]
		routeMetric := fields[3]
		interfaceAlias := strings.Join(fields[4:], " ")

		if !ipv4Regex.MatchString(destinationPrefix) && !ipv6Regex.MatchString(destinationPrefix) {
			continue
		}

		family := "ipv4"
		if strings.Contains(destinationPrefix, ":") {
			family = "ipv6"
		}

		if getOnlyDefaults {
			if family == "ipv4" && destinationPrefix != "0.0.0.0/0" {
				continue
			}
			if family == "ipv6" && destinationPrefix != "::/0" {
				continue
			}
		}

		dest := destinationPrefix
		netmask := ""

		if strings.Contains(destinationPrefix, "/") {
			parts := strings.Split(destinationPrefix, "/")
			dest = parts[0]
			prefixLen, _ := strconv.Atoi(parts[1])

			if family == "ipv4" {
				netmask = cidrToNetmask(prefixLen)
			} else {
				netmask = "/" + parts[1]
			}
		}

		if nextHop == "0.0.0.0" || nextHop == "::" {
			nextHop = "on-link"
		}

		metric, _ := strconv.Atoi(routeMetric)

		routes = append(routes, RouteStruct{
			Interface:   interfaceAlias,
			Destination: dest,
			Netmask:     netmask,
			NextHop:     nextHop,
			Metric:      metric,
			Family:      family,
		})
	}

	return routes, nil
}
