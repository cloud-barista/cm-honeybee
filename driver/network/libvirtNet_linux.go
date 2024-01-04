// Getting libvirt network information for Linux

//go:build linux

package network

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"github.com/libvirt/libvirt-go-xml"
	"net"
	"time"
)

var libvirtSocketFile = "/var/run/libvirt/libvirt-sock"

func libvirtConnect() (*libvirt.Libvirt, error) {
	c, err := net.DialTimeout("unix", libvirtSocketFile, 2*time.Second)
	if err != nil {
		errMsg := fmt.Sprintf("failed to dial libvirt: %v", err)
		logger.Println(logger.ERROR, true, "LIBVIRT_NET: "+errMsg)

		return nil, errors.New(errMsg)
	}

	l := libvirt.NewWithDialer(dialers.NewAlreadyConnected(c))
	if err := l.Connect(); err != nil {
		errMsg := fmt.Sprintf("failed to connect: %v", err)
		logger.Println(logger.ERROR, true, "LIBVIRT_NET: "+errMsg)

		return nil, errors.New(errMsg)
	}

	return l, nil
}

func GetLibvirtNetInfo() (network.LibvirtNet, error) {
	var libvirtNetwork network.LibvirtNet

	if !fileutil.IsExist(libvirtSocketFile) {
		logger.Println(logger.DEBUG, false, "LIBVIRT_NET: libvirt socket file not found.")

		return libvirtNetwork, nil
	}

	l, err := libvirtConnect()
	if err != nil {
		return libvirtNetwork, nil
	}
	defer func() {
		_ = l.Disconnect()
	}()

	domains, _, err := l.ConnectListAllDomains(1, 0)
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve domains: %v", err)
		logger.Println(logger.ERROR, true, "LIBVIRT_NET: "+errMsg)

		return libvirtNetwork, errors.New(errMsg)
	}

	var libvirtDomains []network.LibvirtDomain
	for _, domain := range domains {
		xmlDesc, err := l.DomainGetXMLDesc(domain, 0)
		if err != nil {
			return libvirtNetwork, err
		}

		var domainXML libvirtxml.Domain
		err = domainXML.Unmarshal(xmlDesc)
		if err != nil {
			logger.Printf(logger.ERROR, true,
				"LIBVIRT_NET: Failed to unmarshal XML for domain %s\n", domain.Name)
			continue
		}

		var libvirtDomain network.LibvirtDomain

		libvirtDomain.DomainName = domain.Name
		libvirtDomain.DomainUUID = fmt.Sprintf("%x", domain.UUID)

		if domainXML.Devices != nil {
			libvirtDomain.Interfaces = domainXML.Devices.Interfaces
		}

		libvirtDomains = append(libvirtDomains, libvirtDomain)
	}

	libvirtNetwork.Domains = libvirtDomains

	return libvirtNetwork, nil
}
