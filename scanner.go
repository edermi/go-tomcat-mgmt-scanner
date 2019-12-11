package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// Checks if a port is open. If yes, a new TomcatManagerInstance is created.
// The function returns and the TomcatManagerInstance running in a separate
// goroutine will create work until all creds are checked
func checkPort(target string, port uint, timeout time.Duration) {
	if target == "" {
		return
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", target, port), timeout)
	if err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			prettyPrintLn(warning, "Too many open files. You are running too many scan workers and the OS is limiting file descriptors. YOU ARE MISSING SCAN RESULTS. Scan with less workers")
		} else {
			return
		}
	}
	defer conn.Close()
	// Port is open since no errors occured until here
	prettyPrintLn(info, fmt.Sprintf("%s:%d is open", target, port))
	CheckEndpoint(target, port, false)
}
