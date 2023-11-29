package software

import (
	"fmt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/pkg/errors"
	"net"
	"time"

	"github.com/digitalocean/go-libvirt"
)

func GetLibvirtNetwork() error {
	// c, err := net.DialTimeout("unix", "/var/run/libvirt/libvirt-sock", 2*time.Second)
	c, err := net.DialTimeout("tcp", "192.168.110.240:16509", 2*time.Second)
	if err != nil {
		return errors.Errorf("failed to dial libvirt: %v", err)
	}

	l := libvirt.NewWithDialer(dialers.NewAlreadyConnected(c))
	if err := l.Connect(); err != nil {
		return errors.Errorf("failed to connect: %v", err)
	}

	v, err := l.ConnectGetLibVersion()
	if err != nil {
		return errors.Errorf("failed to retrieve libvirt version: %v", err)
	}
	fmt.Println("libvirt Version:", v)

	//flags := libvirt.ConnectListDomainsActive | libvirt.ConnectListDomainsRunning
	//domains, _, err := l.ConnectListAllDomains(1, flags)
	//if err != nil {
	//	return errors.Errorf("failed to retrieve domains: %v", err)
	//}
	//
	//l.NetworkListAllPorts()

	return nil
}
