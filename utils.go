package main

import (
	"net"
	"strconv"

	"github.com/apparentlymart/go-cidr/cidr"
)

func buildRequestURL(secure bool, host string, port int, managerPath string) string {
	// TODO: Logic in code to check both regardless of port
	if secure {
		return "https://" + host + ":" + strconv.Itoa(port) + managerPath
	}
	return "http://" + host + ":" + strconv.Itoa(port) + managerPath
}

func expandNetwork(network *net.IPNet) []net.IP {
	addrcount := cidr.AddressCount(network)
	ips := make([]net.IP, 0)
	nextHost, _ := cidr.Host(network, 0)
	for hostnum := uint64(0); hostnum < addrcount; hostnum++ {
		ips = append(ips, nextHost)
		nextHost = cidr.Inc(nextHost)
	}
	return ips
}
