package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/fatih/color"
)

// TcInstance represents one possible Tomcat Manager instance
type TcInstance struct {
	host        string
	port        uint
	managerPath string
}

// ScannerConfig holds the global configuration
type ScannerConfig struct {
	targetRange  []net.IP
	ports        []uint
	managerPath  string
	goroutines   uint
	userfile     string
	passfile     string
	userpassfile string
}

// LogType is used to specify if and how information is printed
type LogType int

const (
	debug LogType = iota
	info
	goodnews
	badnews
	profit
	warning
	err
)

var yellow, red, green, cyan func(a ...interface{}) string

func init() {
	yellow = color.New(color.FgYellow).Add(color.Bold).SprintFunc()
	red = color.New(color.FgHiRed).Add(color.Bold).SprintFunc()
	green = color.New(color.FgGreen).Add(color.Bold).SprintFunc()
	cyan = color.New(color.FgCyan).Add(color.Bold).SprintFunc()
}

func prettyPrintLn(logType LogType, msg string) {
	switch logType {
	case debug:
		if Debug {
			log.Printf("%s: %s\n", cyan("[D]"), msg)
		}
	case info:
		log.Printf("%s: %s\n", "[*]", msg)
	case goodnews:
		log.Printf("%s: %s\n", green("[+]"), msg)
	case badnews:
		log.Printf("%s: %s\n", yellow("[-]"), msg)
	case profit:
		log.Printf("%s: %s", green("[$]"), msg)
	case warning:
		log.Printf("%s: %s\n", yellow("[-]"), msg)
	case err:
		log.Printf("%s: %s\n", red("[X]"), msg)
	}
}

func parseCommandLineArgs() ScannerConfig {
	networkUnparsed := flag.String("target", "", "The target network range in CIDR notation, e.g. 10.10.10.0/24")
	portsUnparsed := flag.String("ports", "8080,8443,80,443,8000,8888", "Comma separated list of target ports.")
	managerPathUnparsed := flag.String("managerpath", "/manager/html", "Manager path.")
	goroutines := flag.Uint("concurrency", 100, "Concurrent Goroutines to use. Due to kernel limitations on linux, it should not be more than 'ulimit -n / 7'.")
	randomizeHosts := flag.Bool("randomize", true, "Randomize the order that IP:Port is accessed.")
	userfile := flag.String("userfile", "", "A file containing user names to test. Requires also a passfile. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.")
	passfile := flag.String("passfile", "", "A file containing passwords to test. Requires also a userfile. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.")
	userpassfile := flag.String("userpassfile", "", "A file containing username:password combinations. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.")
	flag.BoolVar(&Debug, "debug", false, "Enable debugging output.")
	flag.Parse()
	ips := parseStringToNetwork(networkUnparsed, randomizeHosts)
	ports := parseStringToPorts(portsUnparsed)
	managerPath := parseStringToManagerPath(managerPathUnparsed)
	sc := ScannerConfig{ips, ports, managerPath, *goroutines, *userfile, *passfile, *userpassfile}
	prettyPrintLn(info, fmt.Sprintf("Ports to scan: %s", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ports)), ","), "[]"))) // ",".join(ports), but with static types... -.-
	prettyPrintLn(info, fmt.Sprintf("Manager path: %s", managerPath))
	prettyPrintLn(info, fmt.Sprintf("Debug is on: %t", Debug))
	prettyPrintLn(debug, fmt.Sprintf("Concurrent goroutines to use: %d", *goroutines))
	prettyPrintLn(debug, fmt.Sprintf("Host order is randomized: %t", *randomizeHosts))
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
		prettyPrintLn(err, "Error parsing IP address!")
		panic("Either IP address is missing or format is wrong. Try e.g. -target 10.0.0.0/8 or -help")
	}
	netRange, _ := strconv.ParseUint(stringSplitted[1], 10, 8)
	var ipArr [4]byte
	for i := uint8(0); i < 4; i++ {
		val, _ := strconv.ParseUint(ipStringSplitted[i], 10, 8)
		ipArr[i] = uint8(val)
	}
	return expandNetwork(&net.IPNet{IP: net.IPv4(ipArr[0], ipArr[1], ipArr[2], ipArr[3]), Mask: net.CIDRMask(int(netRange), 32)}, *randomizeHosts)
}

func buildRequestURL(secure bool, instance *TcInstance) string {
	target := instance.host + ":" + strconv.FormatUint(uint64(instance.port), 10) + instance.managerPath
	if secure {
		return "https://" + target
	}
	return "http://" + target
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
	prettyPrintLn(info, fmt.Sprintf("Completed in %s\n", elapsed))
}
