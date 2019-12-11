# go-tomcat-mgmt-scanner

A small tool to find and brute force tomcat manager logins.

Current project version: 2.0

# About

A simple brute force scanner that tries to identify tomcat manager applications. If such a manager is found, it tries to find valid login credentials by trying common combinations.
The initial wordlists for credential brute forcing are those from Metasploit. The scanner supports userpass files containing `<user>:<password>` entries and files containing solely usernames or passwords which are tried in every possible combination. **There is currently no logic to check if found applications are actually Tomcat manager login pages**. This means that the scanner may also be used to brute force any HTTP auth.

# Build

```
$ go get -u github.com/edermi/go-tomcat-mgmt-scanner
$ make # builds for some architectures/platforms and drops binaries to build/

# or

$ env GOOS=linux GOARCH=arm GOARM=7 go build # Raspberry Pi
$ env GOOS=windows GOARCH=amd64 go build # x64 Windows
$ env GOOS=linux GOARCH=amd64 go build # x64 linux
$ ....
```

# Usage

Example: `./go-tomcat-mgmt-scanner -target 172.16.10.0/24 -ports 80,443 -concurrency 1000 -timeout 2000ms -debug`

Command line options: 

```
  -avoid-lockout
    	Try to avoid lockout by waiting Tomcat's default lockout treshold between tries. Your scan may get suuuper slow, but in the end, success matters.
  -concurrency uint
    	Concurrent Goroutines to use. Due to kernel limitations on linux, it should not be more than 'ulimit -n / 7'. (default 100)
  -debug
    	Enable debugging output.
  -ignoreInsecure
    	Ignore certificate errors. If you only want secure connections, set this to false. (default true)
  -managerpath string
    	Manager path. (default "/manager/html")
  -passfile string
    	A file containing passwords to test. Requires also a userfile. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.
  -ports string
    	Comma separated list of target ports. (default "8080,8443,80,443,8000,8888")
  -target string
    	The target network range in CIDR notation, e.g. 10.10.10.0/24
  -timeout duration
    	HTTP timeout. Specify with unit suffix, e.g. '2500ms' or '3s'. (default 2s)
  -userfile string
    	A file containing user names to test. Requires also a passfile. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.
  -userpassfile string
    	A file containing username:password combinations. If neither user-, password- and userpass list is given, the default lists from Metasploit project are used.
```

# License

See LICENSE. User and password lists are taken from Metasploit project which are licensed BSD 3-clause.

# DISCLAIMER

The usual stuff. Don't do bad things and only use it against targets you are permitted to attack. You are on your own if something breaks.