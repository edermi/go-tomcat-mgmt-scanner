package main

import (
	"fmt"
	"sync"
	"time"
)

// Debug is used for more verbose output messages
var Debug bool

var conf *Configuration
var version = "2.0"
var kudos = "By Michael Eder. https://github.com/edermi, https://twitter.com/michael_eder_"

func main() {
	prettyPrintLn(info, fmt.Sprintf("go-tomcat-mgmt-scanner version %s", version))
	prettyPrintLn(info, kudos)
	conf = parseCommandLineArgs()
	defer timeTrack(time.Now()) // Count execution time

	go generateWork()
	doWork()
}

func generateWork() {
	for {
		ip, e := conf.NextIP()
		if e != nil {
			prettyPrintLn(debug, "Done generating scan work")
			conf.WriteScanQueue() <- conf.ScanFinished
			close(conf.WriteScanQueue())
			break
		}
		for _, port := range conf.AllPorts() {
			t := ip.String()
			p := port
			to := conf.Timeout()
			conf.WriteScanQueue() <- func() {
				checkPort(t, p, to)
			}
		}
	}
}

func doWork() {
	var wg sync.WaitGroup
	for i := uint(0); i < conf.goroutines; i++ {
		wg.Add(1)
		go func(scanQueue <-chan func(), bruteQueue <-chan func()) {
			for {
				if scanQueue == nil && bruteQueue == nil {
					break
				}
				select {
				case scan, ok := <-scanQueue:
					if !ok {
						scanQueue = nil
						continue
					}
					scan()
				case brute, ok := <-bruteQueue:
					if !ok {
						bruteQueue = nil
						continue
					}
					brute()
				}
			}
			wg.Done()
		}(conf.ReadScanQueue(), conf.ReadBruteQueue())
	}
	wg.Wait()
	return
}
