package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
)

// TcInstance represents one possible Tomcat Manager instance
type TcInstance struct {
	host             string
	port             uint
	managerPath      string
	checked          bool
	managerAvailable bool
	finished         bool
}

// ScannerConfig holds the global configuration
type ScannerConfig struct {
	targetRange []net.IP
	ports       []uint
	managerPath string
	goroutines  uint
}

func parseCommandLineArgs() ScannerConfig {
	networkUnparsed := flag.String("target", "", "The target network range in CIDR notation, e.g. 10.10.10.0/24")
	portsUnparsed := flag.String("ports", "8080,8443,80,443,8000,8888", "Comma separated list of target ports.")
	managerPathUnparsed := flag.String("managerpath", "/manager/html", "Manager path.")
	goroutines := flag.Uint("concurrency", 150, "Concurrent Goroutines to use. Due to kernel limitations on linux, it should not be more than 'ulimit -n / 7'.")
	randomizeHosts := flag.Bool("randomize", true, "Randomize the order that IP:Port is accessed.")
	flag.Parse()
	ips := parseStringToNetwork(networkUnparsed, randomizeHosts)
	ports := parseStringToPorts(portsUnparsed)
	managerPath := parseStringToManagerPath(managerPathUnparsed)
	sc := ScannerConfig{ips, ports, managerPath, *goroutines}
	return sc
}

func parseStringToManagerPath(rawString *string) string {
	// Only basic checks to make sure building the request later creates a valid request
	if strings.HasPrefix(*rawString, "/") {
		return *rawString
	}
	return "/" + *rawString
}

func parseStringToPorts(rawString *string) []uint {
	portStringSplitted := strings.Split(*rawString, ",")
	portSlice := make([]uint, 0)
	for i := 0; i < len(portStringSplitted); i++ {
		val, _ := strconv.ParseUint(portStringSplitted[i], 10, 16)
		portSlice = append(portSlice, uint(val))
	}
	return portSlice
}

func parseStringToNetwork(rawString *string, randomizeHosts *bool) []net.IP {
	var stringSplitted []string
	stringSplitted = strings.Split(*rawString, "/")
	ipString := stringSplitted[0]
	ipStringSplitted := strings.Split(ipString, ".")
	if len(ipStringSplitted) != 4 {
		panic("Error parsing IP address")
	}
	netRange, _ := strconv.ParseUint(stringSplitted[1], 10, 8)
	var ipArr [4]byte
	for i := uint8(0); i < 4; i++ {
		val, _ := strconv.ParseUint(ipStringSplitted[i], 10, 8)
		ipArr[i] = uint8(val)
	}
	return expandNetwork(&net.IPNet{IP: net.IPv4(ipArr[0], ipArr[1], ipArr[2], ipArr[3]), Mask: net.CIDRMask(int(netRange), 32)}, *randomizeHosts)
}

func buildRequestURL(secure bool, host string, port uint, managerPath string) string {
	if secure {
		return "https://" + host + ":" + strconv.FormatUint(uint64(port), 10) + managerPath
	}
	return "http://" + host + ":" + strconv.FormatUint(uint64(port), 10) + managerPath
}

func expandNetwork(network *net.IPNet, randomize bool) []net.IP {
	addrcount := cidr.AddressCount(network)
	ips := make([]net.IP, 0)
	nextHost, _ := cidr.Host(network, 0)
	for hostnum := uint64(0); hostnum < addrcount; hostnum++ {
		ips = append(ips, nextHost)
		nextHost = cidr.Inc(nextHost)
	}
	if randomize {
		rand.Seed(time.Now().UTC().UnixNano())
		rand.Shuffle(len(ips), func(i, j int) {
			ips[i], ips[j] = ips[j], ips[i]
		})
	}
	return ips
}

// Used to track how long the execution takes
func timeTrack(start time.Time) {
	elapsed := time.Since(start)
	fmt.Printf("Completed in %s", elapsed)
}
