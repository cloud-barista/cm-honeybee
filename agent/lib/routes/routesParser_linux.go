//go:build linux

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
func GetRoutes(getOnlyDefaults bool) ([]RouteStruct, error) {
	var file = "/proc/net/route"

	f, err := os.Open(file)
	if err != nil {
		errMsg := fmt.Sprintf("can't access %s", file)
		logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
		return nil, errors.New(errMsg)
	}
	defer func() {
		_ = f.Close()
	}()

	readAll, err := io.ReadAll(f)
	if err != nil {
		errMsg := fmt.Sprintf("can't read %s", file)
		logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
		return nil, errors.New(errMsg)
	}

	const (
		sep              = "\t" // field separator
		interfaceField   = 0    // field containing string interface name
		destinationField = 1    // field containing hex destination address
		gatewayField     = 2    // field containing hex gateway address
		metricField      = 6    // field containing int metric
		maskField        = 7    // field containing hex mask
	)
	scanner := bufio.NewScanner(bytes.NewReader(readAll))

	// Skip header line1
	if !scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
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
			logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
			return nil, errors.New(errMsg)
		}

		destinationHex := "0x" + tokens[destinationField]
		destination, err := linuxLittleEndianHexToNetIP(destinationHex)
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
			return nil, errors.New(errMsg)
		}

		netmaskHex := "0x" + tokens[maskField]
		netmask, err := linuxLittleEndianHexToNetIP(netmaskHex)
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
			return nil, errors.New(errMsg)
		}

		nextHopHex := "0x" + tokens[gatewayField]
		nextHop, err := linuxLittleEndianHexToNetIP(nextHopHex)
		if err != nil {
			errMsg := err.Error()
			logger.Println(logger.ERROR, true, "ROUTES: "+errMsg)
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

		metric, _ := strconv.Atoi(tokens[metricField])

		routes = append(routes, RouteStruct{
			Interface:   tokens[interfaceField],
			Destination: destination.String(),
			Netmask:     netmask.String(),
			NextHop:     nextHopStr,
			Metric:      metric,
		})
	}

	return routes, nil
}
