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
	guesses := loadUserPass()
	users := loadUsers()
	passwords := loadPasswords()
	guesses = append(guesses, makeGuesses(users, passwords)...)
	if len(guesses) == 0 {
		prettyPrintLn(err, "There are no user:password combinations loaded, therefore there won't be any brute force logins")
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

func loadUserPass() []Guess {
	var guesses = make([]Guess, 0)
	content := loadFile("./userpass.txt")
	for _, userpass := range content {
		splitted := strings.Split(userpass, ":")
		guesses = append(guesses, Guess{splitted[0], splitted[1]})
	}
	return guesses
}

func loadUsers() []string {
	return loadFile("./users.txt")
}

func loadPasswords() []string {
	return loadFile("./passwords.txt")
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
