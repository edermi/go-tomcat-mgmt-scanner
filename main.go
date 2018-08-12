package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	tc := tcinstance{"localhost", 8081, "/manager/html", false, false, false}
	tc.check()
	guesses := buildGuesses()
	for _, guess := range guesses {
		fmt.Printf("Checking Username: %s Password: %s\n", guess.username, guess.password)
	}
	fmt.Printf("Manager available: %t", tc.managerAvailable)
}

func buildRequestURL(secure bool, host string, port int, managerPath string) string {
	if secure {
		return "https://" + host + ":" + strconv.Itoa(port) + managerPath
	}
	return "http://" + host + ":" + strconv.Itoa(port) + managerPath
}

func (tc *tcinstance) check() {
	client := &http.Client{
		Timeout: time.Second * 15,
	}
	resp, err := client.Get(buildRequestURL(false, tc.host, tc.port, tc.managerPath))
	if err != nil {
		tc.managerAvailable = false
		tc.checked = true
		tc.finished = true
		return
	}
	if resp.StatusCode == 404 {
		tc.managerAvailable = false
	} else if resp.StatusCode == 403 {
		tc.managerAvailable = false
	} else if resp.StatusCode == 401 {
		tc.managerAvailable = true
		fmt.Printf("Manager found at %s:%d%s\n", tc.host, tc.port, tc.managerPath)
	} else {
		fmt.Print(resp.StatusCode)
	}
	tc.checked = true
}

func request(tc *tcinstance, username string, password string) (status int) {
	client := &http.Client{
		Timeout: time.Second * 15,
	}
	req, err := http.NewRequest("GET", buildRequestURL(false, tc.host, tc.port, tc.managerPath), nil)
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return -1
	}
	log.Printf("%d", resp.StatusCode)
	return 0
}
