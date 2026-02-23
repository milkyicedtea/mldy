package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func askYesNo(prompt string) bool {
	return askYesNoDefault(prompt, true)
}

func askYesNoDefaultNo(prompt string) bool {
	return askYesNoDefault(prompt, false)
}

func askYesNoDefault(prompt string, defaultYes bool) bool {
	var input string
	if defaultYes {
		fmt.Printf("%s [Y/n]: ", prompt)
	} else {
		fmt.Printf("%s [y/N]: ", prompt)
	}
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false
	}

	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
