package utils

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// wordList is the cached list of words loaded from the words.txt file
var wordList []string

// loadWordList loads a list of words from a file
func loadWordList(filePath string) ([]string, error) {
	// Check if we've already loaded the words
	if len(wordList) > 0 {
		return wordList, nil
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read word list file: %w", err)
	}

	// Split into words
	words := strings.Split(string(data), "\n")

	// Filter empty lines and trim whitespace
	var cleanWords []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" {
			cleanWords = append(cleanWords, word)
		}
	}

	// Cache the word list
	wordList = cleanWords

	return cleanWords, nil
}

// GeneratePassword creates a secure password using words, numbers, and special characters
func GeneratePassword(wordListPath string) (string, error) {
	// Set random seed
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	// Load word list
	words, err := loadWordList(wordListPath)
	if err != nil {
		return "", fmt.Errorf("failed to load word list: %w", err)
	}

	if len(words) < 5 {
		return "", fmt.Errorf("word list is too short, needs at least 5 words")
	}

	// Special characters to use
	specialChars := "';:-+,.\"\\/?!"

	// Generate a 4-digit number
	num := fmt.Sprintf("%d", r.Intn(8999)+1000)

	// Pick a random special character
	separator := string(specialChars[r.Intn(len(specialChars))])

	// Generate password components
	passwordWords := make([]string, 5)
	for i := 0; i < 5; i++ {
		passwordWords[i] = words[r.Intn(len(words))]
	}

	// Choose a random position to capitalize a word and add the number
	plc := r.Intn(4) // 0-3
	passwordWords[plc], passwordWords[4] = strings.ToUpper(passwordWords[plc]), num

	// Join words with the separator
	password := strings.Join(passwordWords, separator)

	return password, nil
}
