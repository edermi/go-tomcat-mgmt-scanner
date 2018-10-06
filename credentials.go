package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Guess where we store a username:password guess.
type Guess struct {
	username string
	password string
}

func buildGuesses() []Guess {
	var users, passwords []string
	var guesses []Guess
	if scannerConfig.userpassfile != "" {
		guesses = loadUserPass(scannerConfig.userpassfile)
	}
	if scannerConfig.userfile != "" {
		users = loadFile(scannerConfig.userfile)
	}
	if scannerConfig.passfile != "" {
		passwords = loadFile(scannerConfig.passfile)
	}
	guesses = append(guesses, makeGuesses(users, passwords)...)
	if len(guesses) == 0 {
		prettyPrintLn(warning, "There are no custom user:password combinations loaded, using default metasploit combinations")
		guesses = defaultGuesses()
	}
	return guesses
}

func loadFile(filename string) []string {
	var content = make([]string, 0)
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		prettyPrintLn(warning, fmt.Sprintf("File %s not found", filename))
		return content
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if len(scanner.Text()) > 0 {
			content = append(content, scanner.Text())
		}
	}
	return content
}

func loadUserPass(filename string) []Guess {
	var guesses = make([]Guess, 0)
	content := loadFile(filename)
	for _, userpass := range content {
		splitted := strings.Split(userpass, ":")
		guesses = append(guesses, Guess{splitted[0], splitted[1]})
	}
	return guesses
}

func makeGuesses(users, passwords []string) []Guess {
	var guesses = make([]Guess, 0)
	for _, username := range users {
		for _, password := range passwords {
			guesses = append(guesses, Guess{username, password})
		}
	}
	return guesses
}
