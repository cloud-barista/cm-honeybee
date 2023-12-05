// Netfilter is only for linux

//go:build windows

package network

import (
	"errors"
	"github.com/cloud-barista/cm-honeybee/model/network"
)

func GetNetfilterList() (network.Netfilter, error) {
	return network.Netfilter{}, errors.New("getting netfilter information is only for Linux")
}
