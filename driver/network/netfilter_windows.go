// Netfilter is only for linux

//go:build windows

package network

import (
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
)

func GetNetfilterList() (network.Netfilter, error) {
	return network.Netfilter{}, nil
}
