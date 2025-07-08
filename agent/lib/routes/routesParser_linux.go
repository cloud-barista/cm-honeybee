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
	Family      string
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

func parseIPv6Address(hexAddr string) (net.IP, error) {
	if len(hexAddr) != 32 {
		return nil, fmt.Errorf("invalid IPv6 address length: %d", len(hexAddr))
	}

	ip := make(net.IP, 16)
	for i := 0; i < 16; i++ {
		byteStr := hexAddr[i*2 : i*2+2]
		b, err := strconv.ParseUint(byteStr, 16, 8)
		if err != nil {
			return nil, err
		}
		ip[i] = byte(b)
	}

	return ip, nil
}

// getIPv4Routes parses the route file located at /proc/net/route
// and returns the IPv4 address of the default gateway. The default gateway
// is the one with Destination value of 0.0.0.0.
//
// The Linux route file has the following format:
//
// $ cat /proc/net/route
//
// Iface   Destination Gateway     Flags   RefCnt  Use Metric  Mask
// eno1    00000000    C900A8C0    0003    0   0   100 00000000    0   00
// eno1    0000A8C0    00000000    0001    0   0   100 00FFFFFF    0   00
func getIPv4Routes(getOnlyDefaults bool) ([]RouteStruct, error) {
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
			Family:      "ipv4",
		})
	}

	return routes, nil
}

// getIPv6Routes parses the route file located at /proc/net/ipv6_route
// and returns the IPv6 address of the default gateway. The default gateway
// is the one with Destination value of 00000000000000000000000000000000.
//
// The Linux route file has the following format:
//
// $ cat /proc/net/ipv6_route
//
// 00000000000000000000000000000001 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 00000002 00000000 00000001       lo
// 20010000000000000000000000000000 40 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 00000001 00000000 00000001     eth1
// fe800000000000000000000000000000 40 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 00000001 00000000 00000001     eth1
// fe800000000000000000000000000000 40 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 00000001 00000000 00000001     eth0
// 00000000000000000000000000000000 00 00000000000000000000000000000000 00 20010000000000000000000000000010 00000400 00000001 00000000 00000003     eth1
// 00000000000000000000000000000001 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000000 00000008 00000000 80200001       lo
// 20010000000000000000000000000001 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000000 00000002 00000000 80200001     eth1
// fe80000000000000f8163efffe7eaa58 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000000 00000003 00000000 80200001     eth1
// fe80000000000000f8163efffe967a72 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000000 00000002 00000000 80200001     eth0
// ff000000000000000000000000000000 08 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 0000000a 00000000 00000001     eth1
// ff000000000000000000000000000000 08 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 00000009 00000000 00000001     eth0
// 00000000000000000000000000000000 00 00000000000000000000000000000000 00 00000000000000000000000000000000 ffffffff 00000001 00000000 00200200       lo
func getIPv6Routes(getOnlyDefaults bool) ([]RouteStruct, error) {
	var file = "/proc/net/ipv6_route"

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

	scanner := bufio.NewScanner(bytes.NewReader(readAll))
	var routes []RouteStruct

	for scanner.Scan() {
		row := scanner.Text()
		fields := strings.Fields(row)

		if len(fields) < 10 {
			continue
		}

		destAddr := fields[0]
		prefixLen, _ := strconv.ParseInt(fields[1], 16, 32)
		nextHopAddr := fields[4]
		metric, _ := strconv.ParseInt(fields[5], 16, 32)
		interfaceName := fields[9]

		destination, err := parseIPv6Address(destAddr)
		if err != nil {
			continue
		}

		nextHop, err := parseIPv6Address(nextHopAddr)
		if err != nil {
			continue
		}

		if getOnlyDefaults && destAddr != "00000000000000000000000000000000" {
			continue
		}

		nextHopStr := nextHop.String()
		if nextHop.String() == "::" {
			nextHopStr = "on-link"
		}

		netmask := fmt.Sprintf("/%d", prefixLen)

		routes = append(routes, RouteStruct{
			Interface:   interfaceName,
			Destination: destination.String(),
			Netmask:     netmask,
			NextHop:     nextHopStr,
			Metric:      int(metric),
			Family:      "ipv6",
		})
	}

	return routes, nil
}

func GetRoutes(getOnlyDefaults bool) ([]RouteStruct, error) {
	var allRoutes []RouteStruct

	ipv4Routes, err := getIPv4Routes(getOnlyDefaults)
	if err != nil {
		return nil, err
	}
	allRoutes = append(allRoutes, ipv4Routes...)

	ipv6Routes, err := getIPv6Routes(getOnlyDefaults)
	if err != nil {
		logger.Println(logger.WARN, true, "ROUTES: IPv6 routes not available: "+err.Error())
	} else {
		allRoutes = append(allRoutes, ipv6Routes...)
	}

	return allRoutes, nil
}
