package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func buildGuesses(userfile, passfile, userpassfile string) []Guess {
	var users, passwords []string
	var guesses []Guess
	if userpassfile != "" {
		guesses = loadUserPass(userpassfile)
	}
	if userfile != "" {
		users = loadFile(userfile)
	}
	if passfile != "" {
		passwords = loadFile(passfile)
	}
	guesses = append(guesses, makeGuesses(users, passwords)...)
	if len(guesses) == 0 {
		prettyPrintLn(warning, "There are no custom user:password combinations loaded, using default metasploit combinations")
		guesses = defaultGuesses()
	}
	prettyPrintLn(info, fmt.Sprintf("%d user:password combinations loaded", len(guesses)))
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
