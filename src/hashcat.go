package main

import (
	"crypto/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func InvokeHashcatBruteForce(cfg *Config, mask string, extraArgs []string) error {
	args := []string{"-a", "3", mask}
	if extraArgs != nil && len(extraArgs) > 1 {
		args = append(args, extraArgs...)
	}
	return InvokeHashcat(cfg, args)
}

func InvokeHashcatStackingRules(cfg *Config, extraArgs []string) error {
	args := []string{"-a", "0"}
	if extraArgs == nil || len(extraArgs) < 1 {
		return FmtError("No extra arguments provided for stacking rules phase")
	}
	args = append(args, extraArgs...)
	return InvokeHashcat(cfg, args)
}

func InvokeHashcatDictWithRule(cfg *Config, dictionary string, rule string) error {
	args := []string{"-a", "0", dictionary}
	if rule != "" {
		args = append(args, "-r", rule)
	}
	return InvokeHashcat(cfg, args)
}

func InvokeHashcat(cfg *Config, extraArgs []string) error {
	randomSession := "Session_" + rand.Text()
	defer func(cfg *Config, randomSession string) {
		LogInfo("Cleaning up hashcat session %s", randomSession)
		LogInfo("Removing hashcat session %s", cfg.HashcatDir+randomSession)
		os.RemoveAll(cfg.HashcatDir + randomSession)
		sessions, _ := filepath.Glob(cfg.HashcatDir + randomSession + "*")
		for _, ss := range sessions {
			LogInfo("Removing hashcat session %s", ss)
			os.RemoveAll(ss)
		}
	}(cfg, randomSession)
	args := []string{"--session=" + randomSession, "-O", "--force", "-w", "4", "-m", cfg.HashType, cfg.HashFile}
	if extraArgs != nil && len(extraArgs) > 1 {
		args = append(args, extraArgs...)
	}

	// on Windows, chdir into hashcat folder then rollback
	if runtime.GOOS == "windows" {
		//get current cwdir
		cwd, err := os.Getwd()
		if err != nil {
			return FmtError("Error getting current working directory:\n%v", err)
		}
		defer os.Chdir(cwd)
		if err := os.Chdir(cfg.HashcatDir); err != nil {
			return FmtError("Error changing directory to %s:\n%v", cfg.HashcatDir, err)
		}
	}

	LogInfo("\n\033[42m***************************************************************************************************\nRunning hashcat phase %s with %v\033[0m", cfg.Phase, args)
	cmd := exec.Command(cfg.HashcatBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	// if exit status 1 => ignore
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return nil
			}
		}
	}
	return err
}
