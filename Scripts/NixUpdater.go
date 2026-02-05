package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const logFile = "/var/log/nixos-update.log"

func run(cmd string, args ...string) (string, error) {
	c := exec.Command(cmd, args...)
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	err := c.Run()
	return out.String(), err
}

func getCurrentGeneration() (string, error) {
	out, err := run("nixos-rebuild", "list-generations")
	if err != nil {
		return "", err
	}

	lines := strings.Split(out, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "current") {
			return lines[i], nil
		}
	}
	return "unknown", nil
}

func getPackageList() (string, error) {
	return run("nix-env", "-q", "--installed")
}

func logChange(text string) {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("\n[%s]\n%s\n", timestamp, text))
}

func main() {
	fmt.Println("== NixOS Auto-Updater ==")

	beforeGen, _ := getCurrentGeneration()
	beforePkgs, _ := getPackageList()

	fmt.Println("Running nixos-rebuild switch...")
	out, err := run("nixos-rebuild", "switch")

	if err != nil {
		logChange("UPDATE FAILED:\n" + out)
		fmt.Println("Update failed, logged.")
		return
	}

	afterGen, _ := getCurrentGeneration()
	afterPkgs, _ := getPackageList()

	var diff bytes.Buffer
	diff.WriteString("=== SYSTEM UPDATED ===\n")
	diff.WriteString("Before Generation:\n" + beforeGen + "\n")
	diff.WriteString("After Generation:\n" + afterGen + "\n")

	diff.WriteString("\n=== PACKAGE DIFF ===\n")

	beforeSet := make(map[string]bool)
	for _, line := range strings.Split(beforePkgs, "\n") {
		beforeSet[line] = true
	}

	for _, pkg := range strings.Split(afterPkgs, "\n") {
		if !beforeSet[pkg] && pkg != "" {
			diff.WriteString("+ " + pkg + "\n")
		}
	}

	diff.WriteString("\n=== RAW REBUILD LOG ===\n")
	diff.WriteString(out)

	logChange(diff.String())

	fmt.Println("Update complete. Changes logged.")
}
