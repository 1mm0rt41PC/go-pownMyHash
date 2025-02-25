package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

func updateHistoricalDict(cfg *Config, newPasswords []string) error {
	LogInfo("Updating historical dictionary (%s) with new passwords", cfg.HistoricalDict)
	var existingPasswords []string
	if file, err := os.Open(cfg.HistoricalDict); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			existingPasswords = append(existingPasswords, scanner.Text())
		}
		file.Close()
	}

	allPasswords := append(existingPasswords, newPasswords...)
	sort.Strings(allPasswords)
	uniquePasswords := removeDuplicates(allPasswords)

	return os.WriteFile(cfg.HistoricalDict, []byte(strings.Join(uniquePasswords, "\n")), 0644)
}

func exportFoundPasswords(cfg *Config) uint64 {
	LogInfo("Generate custom dict (%s) from potfile...", cfg.CustomDict)

	// if potfile size doesn't changed from previous => return last count
	if potfile, err := os.Stat(cfg.HashcatPotfile); err == nil {
		if potfile.Size() == cfg.HashcatPotfileSize {
			LogInfo("Potfile size didn't change, skipping custom dict generation")
			return cfg.HashcatFound
		}
		cfg.HashcatPotfileSize = potfile.Size()
	} else {
		LogError("Error getting potfile size:\n%v", err)
		return 0
	}

	// purge custom dict
	os.WriteFile(cfg.CustomDict, []byte(""), 0644)

	// on Windows, chdir into hashcat folder then rollback
	if runtime.GOOS == "windows" {
		//get current cwdir
		cwd, err := os.Getwd()
		if err != nil {
			LogError("Error getting current working directory:\n%v", err)
			return 0
		}
		defer os.Chdir(cwd)
		if err := os.Chdir(cfg.HashcatDir); err != nil {
			LogError("Error changing directory to %s:\n%v", cfg.HashcatDir, err)
			return 0
		}
	}

	LogDebug("Running %s --show -m %s %s", cfg.HashcatBin, cfg.HashType, cfg.HashFile)
	cmd := exec.Command(cfg.HashcatBin, "--show", "--outfile-format=2", "-m", cfg.HashType, cfg.HashFile)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running hashcat show: %v\n", err)
		return 0
	}

	var passwords []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		row := scanner.Text()
		if row == "" || row == "[notfound]" {
			continue
		}
		passwords = append(passwords, row)
	}

	sort.Strings(passwords)
	uniquePasswords := removeDuplicates(passwords)

	if err = os.WriteFile(cfg.CustomDict, []byte(strings.Join(uniquePasswords, "\n")), 0644); err != nil {
		LogError("Error writing custom dict:\n%v", err)
		return 0
	}

	if err = updateHistoricalDict(cfg, uniquePasswords); err != nil {
		LogError("Error updating historical dictionary:\n%v", err)
		return 0
	}

	count := uint64(len(uniquePasswords))
	LogInfo("Found %d new passwords", count)

	if historical, err := os.ReadFile(cfg.HistoricalDict); err == nil {
		historicalCount := len(strings.Split(string(historical), "\n"))
		LogInfo("Historical dictionary (%s) contains %d passwords", cfg.HistoricalDict, historicalCount)
	}
	cfg.HashcatFound = count
	return count
}

func removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
