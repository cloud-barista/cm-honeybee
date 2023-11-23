package routes

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/jollaman999/utils/logger"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type RouteStruct struct {
	Interface   string
	Destination string
	Netmask     string
	NextHop     string
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
func GetWindowsRoutes(getOnlyDefaults bool) ([]RouteStruct, error) {
	routeCmd := exec.Command("route", "print", "0.0.0.0")
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		errMsg := err.Error()
		logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
		return nil, errors.New(errMsg)
	}

	const (
		destinationField = 0 // field containing string dotted destination IP address
		netmaskField     = 1 // field containing string dotted netmask
		gatewayField     = 2 // field containing string dotted gateway IP address
		interfaceField   = 3 // field containing string dotted interface IP address
	)

	ipRegex := regexp.MustCompile(`^(((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4})`)
	var routes []RouteStruct
	lines := strings.Split(string(output), "\n")
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
				logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
				return nil, errors.New(errMsg)
			}

			// found default route
			if getOnlyDefaults && fields[0] != "0.0.0.0" {
				continue
			}

			routes = append(routes, RouteStruct{
				Destination: fields[destinationField],
				Netmask:     fields[netmaskField],
				NextHop:     strings.ToLower(fields[gatewayField]),
				Interface:   fields[interfaceField],
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
		logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
		return nil, errors.New(errMsg)
	}

	return routes, nil
}

func linuxLittleEndianHexToNetIP(hex string) (net.IP, error) {
	netIP := make(net.IP, 4)

	// cast hex address to uint32
	d, err := strconv.ParseInt(hex, 0, 64)
	if err != nil {
		return nil, err
	}
	d32 := uint32(d)

	binary.LittleEndian.PutUint32(netIP, d32)

	return netIP, nil
}

// getLinuxGateways parses the route file located at /proc/net/route
// and returns the IP address of the default gateway. The default gateway
// is the one with Destination value of 0.0.0.0.
//
// The Linux route file has the following format:
//
// $ cat /proc/net/route
//
// Iface   Destination Gateway     Flags   RefCnt  Use Metric  Mask
// eno1    00000000    C900A8C0    0003    0   0   100 00000000    0   00
// eno1    0000A8C0    00000000    0001    0   0   100 00FFFFFF    0   00
func GetLinuxRoutes(getOnlyDefaults bool) ([]RouteStruct, error) {
	var file = "/proc/net/route"

	f, err := os.Open(file)
	if err != nil {
		errMsg := fmt.Sprintf("can't access %s", file)
		logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
		return nil, errors.New(errMsg)
	}
	defer func() {
		_ = f.Close()
	}()

	readAll, err := io.ReadAll(f)
	if err != nil {
		errMsg := fmt.Sprintf("can't read %s", file)
		logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
		return nil, errors.New(errMsg)
	}

	const (
		sep              = "\t" // field separator
		interfaceField   = 0    // field containing string interface name
		destinationField = 1    // field containing hex destination address
		gatewayField     = 2    // field containing hex gateway address
		maskField        = 7    // field containing hex mask
	)
	scanner := bufio.NewScanner(bytes.NewReader(readAll))

	// Skip header line
	if !scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
			return nil, errors.New(errMsg)
		}

		return []RouteStruct{}, nil
	}

	var routes []RouteStruct

	for scanner.Scan() {
		row := scanner.Text()
		tokens := strings.Split(row, sep)
		if len(tokens) < 11 {
			errMsg := "invalid file format of " + file
			logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
			return nil, errors.New(errMsg)
		}

		destinationHex := "0x" + tokens[destinationField]
		destination, err := linuxLittleEndianHexToNetIP(destinationHex)
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
			return nil, errors.New(errMsg)
		}

		netmaskHex := "0x" + tokens[maskField]
		netmask, err := linuxLittleEndianHexToNetIP(netmaskHex)
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
			return nil, errors.New(errMsg)
		}

		nextHopHex := "0x" + tokens[gatewayField]
		nextHop, err := linuxLittleEndianHexToNetIP(nextHopHex)
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "GATEWAY: "+errMsg)
			return nil, errors.New(errMsg)
		}

		// The default interface is the one that's 0 for both destination and mask.
		if getOnlyDefaults && (tokens[destinationField] != "00000000" || tokens[maskField] != "00000000") {
			continue
		}

		nextHopStr := nextHop.String()
		if nextHop.String() == "0.0.0.0" {
			nextHopStr = "on-link"
		}

		routes = append(routes, RouteStruct{
			Interface:   tokens[interfaceField],
			Destination: destination.String(),
			Netmask:     netmask.String(),
			NextHop:     nextHopStr,
		})
	}

	return routes, nil
}
