package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Debug is used for more verbose output messages
var Debug bool

var guesses []Guess
var scannerConfig ScannerConfig
var successfullLogins map[TcInstance]Guess
var version = "1.1.0"
var kudos = "By Michael Eder. @edermi on Github, @michael_eder_ on Twitter."

func main() {
	prettyPrintLn(info, fmt.Sprintf("go-tomcat-mgmt-scanner version %s", version))
	prettyPrintLn(info, kudos)
	scannerConfig = parseCommandLineArgs()
	defer timeTrack(time.Now()) // Count execution time
	guesses = buildGuesses()
	workQueue := make(chan TcInstance, 20) // This channel is used to pass work to the goroutines
	var wg sync.WaitGroup                  // The WaitGroup is used to wait for all goroutines to finish at the end
	successfullLogins = make(map[TcInstance]Guess)
	spawnWorkers(&wg, workQueue)
	fillQueue(workQueue)
	wg.Wait() // Wait for goroutines to finish
	for key := range successfullLogins {
		prettyPrintLn(profit, fmt.Sprintf("%s:%s at %s", successfullLogins[key].username, successfullLogins[key].password, (key.host+":"+strconv.FormatUint(uint64(key.port), 10)+key.managerPath)))
	}
}

// This function fills the workQueue channel with tcinstances that are scanned
// It also tracks process and reports status from time to time
func fillQueue(workQueue chan<- TcInstance) {
	numTargets := len(scannerConfig.targetRange) * len(scannerConfig.ports)
	progress := 0
	tenths := 1
	for {
		element, ok := <-scannerConfig.targetRange
		if !ok {
			break
		}
		prettyPrintLn(debug, fmt.Sprintf("Now sending to %v\n", element))
		for _, port := range scannerConfig.ports {
			if progress > tenths*(numTargets/10) {
				prettyPrintLn(info, fmt.Sprintf("~%d0%% (%d/%d)", tenths, progress, numTargets))
				tenths++
			}
			tc := TcInstance{element.String(), port, scannerConfig.managerPath}
			workQueue <- tc
			progress++
		}
	}
	close(workQueue) // The closed channel also tells the goroutines to return
}

// spawnWorkers creates the desired number of goroutines, adds them to the WaitGroup and receives work from the channel
func spawnWorkers(wg *sync.WaitGroup, workQueue <-chan TcInstance) {
	for worker := uint(0); worker < scannerConfig.goroutines; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{ // We need to initialise our own http client struct
				Timeout: time.Second * 5, // in order to define our own timeout
			}
			for {
				tc, ok := <-workQueue
				if ok == false { // If the channel is closed, we are done
					break
				}
				if tc.check(client, false) {
					for _, guess := range guesses {
						success, _ := request(&tc, &guess)
						if success {
							break
						}
					}
				}
			}
		}()
	}
}

func (tc *TcInstance) check(client *http.Client, TLSenabled bool) (managerAvailable bool) {
	var resp *http.Response
	var err error
	if TLSenabled {
		resp, err = client.Get(buildRequestURL(true, tc))
	} else {
		resp, err = client.Get(buildRequestURL(false, tc))
	}
	if err != nil {
		prettyPrintLn(debug, err.Error()) // For debugging socket issues
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		prettyPrintLn(badnews, fmt.Sprintf("Manager not found at %s:%d%s", tc.host, tc.port, tc.managerPath))
		return false
	} else if resp.StatusCode == http.StatusForbidden {
		prettyPrintLn(badnews, fmt.Sprintf("Manager not found at %s:%d%s", tc.host, tc.port, tc.managerPath))
		return false
	} else if resp.StatusCode == http.StatusUnauthorized {
		prettyPrintLn(goodnews, fmt.Sprintf("Manager found at %s:%d%s", tc.host, tc.port, tc.managerPath))
		return true
	} else if resp.StatusCode == http.StatusBadRequest {
		return tc.check(client, true)
	} else {
		prettyPrintLn(info, fmt.Sprintf("HTTP %d without authentication at %s:%d%s", resp.StatusCode, tc.host, tc.port, tc.managerPath))
		return false
	}
}

func request(tc *TcInstance, guess *Guess) (success bool, err error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	url := buildRequestURL(false, tc)
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(guess.username, guess.password)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		prettyPrintLn(profit, fmt.Sprintf("Success! %s:%s on %s", guess.username, guess.password, url))
		successfullLogins[*tc] = *guess
		return true, nil
	}
	return false, nil
}
