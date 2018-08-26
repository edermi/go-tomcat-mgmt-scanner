package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

var guesses []Guess
var scannerConfig ScannerConfig

func main() {
	scannerConfig = parseCommandLineArgs()
	defer timeTrack(time.Now()) // Count execution time
	guesses = buildGuesses()
	workQueue := make(chan TcInstance, 20) // This channel is used to pass work to the goroutines
	var wg sync.WaitGroup                  // The WaitGroup is used to wait for all goroutines to finish at the end
	spawnWorkers(&wg, workQueue)
	fillQueue(workQueue)
	wg.Wait() // Wait for goroutines to finish
}

// This function fills the workQueue channel with tcinstances that are scanned
// It also tracks process and reports status from time to time
func fillQueue(workQueue chan<- TcInstance) {
	numTargets := len(scannerConfig.targetRange) * len(scannerConfig.ports)
	progress := 0
	tenths := 1
	for _, element := range scannerConfig.targetRange {
		for _, port := range scannerConfig.ports {
			if progress > tenths*(numTargets/10) {
				log.Printf("[*] ~%d0%% (%d/%d)\n", tenths, progress, numTargets)
				tenths++
			}
			tc := TcInstance{element.String(), port, scannerConfig.managerPath, false, false, false}
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
				Timeout: time.Second * 3, // in order to define our own timeout
			}
			for {
				tc, ok := <-workQueue
				if ok == false { // If the channel is closed, we are done
					break
				}
				tc.check(client, false)
				for _, guess := range guesses {
					success, _ := request(&tc, guess.username, guess.password)
					if success {
						break
					}
				}
			}
		}()
	}
}

func (tc *TcInstance) check(client *http.Client, TLSenabled bool) {
	//log.Printf("[-] Checking %s\n", buildRequestURL(false, tc.host, tc.port, tc.managerPath))
	var resp *http.Response
	var err error
	if TLSenabled {
		resp, err = client.Get(buildRequestURL(true, tc.host, tc.port, tc.managerPath))
	} else {
		resp, err = client.Get(buildRequestURL(false, tc.host, tc.port, tc.managerPath))
	}
	if err != nil {
		//log.Println(err.Error()) // For debugging socket issues
		tc.checked = true
		tc.finished = true
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[-] Manager not found at %s:%d%s\n", tc.host, tc.port, tc.managerPath)
		tc.managerAvailable = false
	} else if resp.StatusCode == http.StatusForbidden {
		log.Printf("[-] Manager not found at %s:%d%s\n", tc.host, tc.port, tc.managerPath)
		tc.managerAvailable = false
	} else if resp.StatusCode == http.StatusUnauthorized {
		tc.managerAvailable = true
		log.Printf("[+] Manager found at %s:%d%s\n", tc.host, tc.port, tc.managerPath)
	} else if resp.StatusCode == http.StatusBadRequest {
		tc.check(client, true)
	} else {
		log.Printf("[*] HTTP %d at %s:%d%s\n", resp.StatusCode, tc.host, tc.port, tc.managerPath)
	}
	tc.checked = true
}

func request(tc *TcInstance, username string, password string) (success bool, err error) {
	if !tc.managerAvailable {
		return false, nil
	}
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	url := buildRequestURL(false, tc.host, tc.port, tc.managerPath)
	log.Printf("Requesting %s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		log.Printf("[$] Success! %s:%s on %s\n", username, password, url)
		return true, nil
	}
	return false, nil
}
