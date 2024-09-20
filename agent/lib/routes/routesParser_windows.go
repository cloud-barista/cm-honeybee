//go:build windows

package routes

import (
	"errors"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/logger"
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
}

// Windows route output format is always like this:
// ===========================================================================
// Interface List
// 8 ...00 12 3f a7 17 ba ...... Intel(R) PRO/100 VE Network Connection
// 1 ........................... Software Loopback Interface 1
// ===========================================================================
// IPv4 Route Table
// ===========================================================================
// Active Routes:
// Network Destination        Netmask          Gateway       Interface  Metric
//
//	0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     20
//
// ===========================================================================
//
// Windows commands are localized, so we can't just look for "Active Routes:" string
// I'm trying to pick the active route,
// then jump 2 lines and get the row
// Not using regex because output is quite standard from Windows XP to 8 (NEEDS TESTING)
//
// If multiple default gateways are present, then the one with the lowest metric is returned.
func GetRoutes(getOnlyDefaults bool) ([]RouteStruct, error) {
	output, err := cmd.RunCMD("route print")
	if err != nil {
		errMsg := err.Error()
		logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
		return nil, errors.New(errMsg)
	}

	const (
		destinationField = 0 // field containing string dotted destination IP address
		netmaskField     = 1 // field containing string dotted netmask
		gatewayField     = 2 // field containing string dotted gateway IP address
		interfaceField   = 3 // field containing string dotted interface IP address
		metricField      = 4 // field containing string metric
	)

	ipRegex := regexp.MustCompile(`^(((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4})`)
	var routes []RouteStruct
	lines := strings.Split(output, "\n")
	sep := 0
	for idx, line := range lines {
		if sep == 3 {
			// We just entered the 2nd section containing "Active Routes:"
			if len(lines) <= idx+2 {
				return []RouteStruct{}, nil
			}

			inputLine := lines[idx+2]
			if strings.HasPrefix(inputLine, "=======") {
				// End of routes
				break
			}
			fields := strings.Fields(inputLine)
			if len(fields) < 5 || !ipRegex.MatchString(fields[0]) {
				errMsg := "invalid filed found"
				logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
				return nil, errors.New(errMsg)
			}

			// found default route
			if getOnlyDefaults && (fields[destinationField] != "0.0.0.0" || fields[netmaskField] != "0.0.0.0") {
				continue
			}

			metric, _ := strconv.Atoi(fields[metricField])

			routes = append(routes, RouteStruct{
				Interface:   fields[interfaceField],
				Destination: fields[destinationField],
				Netmask:     fields[netmaskField],
				NextHop:     strings.ToLower(fields[gatewayField]),
				Metric:      metric,
			})
		}
		if strings.HasPrefix(line, "=======") {
			sep++
			continue
		}
	}

	if sep == 0 {
		// We saw no separator lines, so input must have been garbage.
		errMsg := "got invalid result from route command"
		logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
		return nil, errors.New(errMsg)
	}

	return routes, nil
}
