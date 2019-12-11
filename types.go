package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/fatih/color"
)

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

// Guess contains a single username and password to try
type Guess struct {
	username string
	password string
}

// Configuration is global, allowing goroutine-safe access
// to information like default credentials
type Configuration struct {
	networkIterator  *NetIterator
	scanQueue        chan func()
	bruteQueue       chan func()
	ports            []uint // Which ports to check
	managerPath      string // Which path to check
	goroutines       uint
	guesses          []Guess // username:password tuple
	timeout          time.Duration
	avoidLockout     bool
	semaphore        sync.RWMutex
	HTTPClient       http.Client
	scanFinished     bool
	currentlyBruting uint
}

// NextIP fetches the next ip and returns it.
// Do a full lock, since state is changed.
func (conf *Configuration) NextIP() (net.IP, error) {
	conf.semaphore.Lock()
	defer conf.semaphore.Unlock()
	return conf.networkIterator.NextIP()
}

// PortToCheck returns the n-th port to check
func (conf *Configuration) PortToCheck(n uint) (uint, error) {
	conf.semaphore.RLock()
	defer conf.semaphore.RUnlock()
	if n >= uint(len(conf.ports)) {
		return 0, fmt.Errorf("Out of bounds")
	}
	return conf.ports[n], nil
}

// AllPorts returns the complete port list. Used for initial port scanning.
func (conf *Configuration) AllPorts() []uint {
	conf.semaphore.RLock()
	defer conf.semaphore.RUnlock()
	return conf.ports
}

// ManagerPath returns where to look for the manager
func (conf *Configuration) ManagerPath() string {
	conf.semaphore.RLock()
	defer conf.semaphore.RUnlock()
	return conf.managerPath
}

// GuessToCheck returns the n-th Guess
func (conf *Configuration) GuessToCheck(n uint) (Guess, error) {
	conf.semaphore.RLock()
	defer conf.semaphore.RUnlock()
	if n >= uint(len(conf.guesses)) {
		return Guess{"", ""}, fmt.Errorf("Out of bounds")
	}
	return conf.guesses[n], nil
}

// Timeout returns the timeout. goroutine safe.
func (conf *Configuration) Timeout() time.Duration {
	conf.semaphore.RLock()
	defer conf.semaphore.RUnlock()
	return conf.timeout
}

// AvoidLockout returns to avoid lockouts. goroutine safe.
func (conf *Configuration) AvoidLockout() bool {
	conf.semaphore.RLock()
	defer conf.semaphore.RUnlock()
	return conf.avoidLockout
}

// ReadScanQueue returns a receive-only version of the channel
func (conf *Configuration) ReadScanQueue() <-chan func() {
	return conf.scanQueue
}

// WriteScanQueue returns a send-only version of the channel
func (conf *Configuration) WriteScanQueue() chan<- func() {
	return conf.scanQueue
}

// ReadBruteQueue returns a receive-only version of the channel
func (conf *Configuration) ReadBruteQueue() <-chan func() {
	return conf.bruteQueue
}

// WriteBruteQueue returns a send-only version of the channel
func (conf *Configuration) WriteBruteQueue() chan<- func() {
	return conf.bruteQueue
}

// ScanFinished is queued after the last scan task to mark the end of
// the scanning phase
func (conf *Configuration) ScanFinished() {
	time.Sleep(2 * conf.Timeout())
	conf.semaphore.Lock()
	conf.scanFinished = true
	conf.semaphore.Unlock()
	prettyPrintLn(info, fmt.Sprintf("Discovery finished, now there are just %d bruters still running", conf.NumBruters()))
	conf.semaphore.Lock()
	defer conf.semaphore.Unlock()
	if conf.currentlyBruting == 0 {
		close(conf.bruteQueue)
	}
}

// RegisterBruter increments the count of active brutes which is required to know when to shut down
func (conf *Configuration) RegisterBruter() {
	conf.semaphore.Lock()
	defer conf.semaphore.Unlock()
	// We could also do this using sync.atomic, but in unregister
	// there also needs to be checked if scan is done which requires
	// locking, therefore also just lock here
	conf.currentlyBruting++
}

// UnregisterBruter decrements the count of active brutes which is required to know when to shut down
func (conf *Configuration) UnregisterBruter() {
	conf.semaphore.Lock()
	defer conf.semaphore.Unlock()
	conf.currentlyBruting--
	if conf.scanFinished && conf.currentlyBruting == 0 {
		close(conf.bruteQueue)
	}
}

// NumBruters returns the number of currently active bruters
func (conf *Configuration) NumBruters() uint {
	conf.semaphore.RLock()
	defer conf.semaphore.RUnlock()
	return conf.currentlyBruting
}

// NewConfiguration initializes a config
func NewConfiguration(networkIterator *NetIterator, ports []uint, guesses []Guess, managerPath string, goroutines uint, timeout time.Duration, avoidLockout bool, ignoreInsecure bool) *Configuration {
	var tr *http.Transport
	if ignoreInsecure {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	httpClient := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
	c := &Configuration{
		networkIterator:  networkIterator,
		scanQueue:        make(chan func(), int(2*goroutines)), // There should always be place for enough work
		bruteQueue:       make(chan func(), int(2*goroutines)), // There should always be place for enough work
		ports:            ports,
		managerPath:      managerPath,
		goroutines:       goroutines,
		guesses:          guesses,
		timeout:          timeout,
		avoidLockout:     avoidLockout,
		HTTPClient:       *httpClient,
		scanFinished:     false,
		currentlyBruting: 0,
	}
	return c
}

// Bruter is created each time a check was successful.
// It keeps track of guesses and generates more work.
type Bruter struct {
	target       string // hostname or IP
	port         uint
	currentGuess uint
	timeout      time.Duration
	avoidLockout bool
}

func (bruter *Bruter) generateWork() {
	conf.RegisterBruter()
	defer conf.UnregisterBruter()
	prettyPrintLn(info, "Bruter created")
	ctr := 0
	tries := 3                       // TODO: Configurable
	lockoutTime := time.Second * 300 // TODO: Configurable
	// lockoutTime needs to cope with the fact that we just append to a work queue
	// In order to be safe, wait some additional time, e.g. two extra timeouts
	// (queues are sized in such a way that they should be handled once then)
	lockoutTime = lockoutTime + 2*conf.Timeout()
	for ; true; <-time.Tick(lockoutTime) {
		for i := 0; i < tries; i++ {
			guess, e := conf.GuessToCheck(uint(ctr))
			if e != nil {
				return
			}
			conf.bruteQueue <- func() { BruteEndpoint(bruter.target, bruter.port, false, guess) }
			ctr++
		}
	}
}

// NewBruter generates a Bruter and starts the asynchronous work generation
func NewBruter(host string, port uint) {
	bruter := Bruter{
		target:       host,
		port:         port,
		timeout:      conf.Timeout(),
		avoidLockout: conf.AvoidLockout(),
	}
	bruter.generateWork()
}

// NetIterator is a convenience type wrapping the state
// of the random number generator and giving the next IP
// on request. Also counts
type NetIterator struct {
	network   *net.IPNet
	c         *cycle
	current   *uint64
	count     uint64
	end       uint64
	semaphore sync.Mutex
}

// NextIP returns the next IP or an error when done
func (n *NetIterator) NextIP() (net.IP, error) {
	n.semaphore.Lock()
	defer n.semaphore.Unlock()
	ip, e := cidr.Host(n.network, int(*n.current))
	if e != nil {
		return nil, e
	}
	if n.count > n.end {
		return ip, fmt.Errorf("Done")
	}
	n.count++
	next(n.c, n.current)
	return ip, nil
}

// NewNetIterator initializes the address generator
func NewNetIterator(network string) (*NetIterator, error) {
	_, ipnet, e := net.ParseCIDR(network)
	if e != nil {
		prettyPrintLn(err, "Error parsing IP address!")
		panic("Either IP address is missing or format is wrong. Try e.g. -target 10.0.0.0/8 or -help")
	}
	group := getGroup(cidr.AddressCount(ipnet))
	cycle := makeCycle(group, time.Now().UTC().UnixNano())
	start := first(&cycle)
	n := NetIterator{
		network: ipnet,
		c:       &cycle,
		current: &start,
		count:   0,
		end:     cidr.AddressCount(ipnet),
	}
	return &n, nil
}
