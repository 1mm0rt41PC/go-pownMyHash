package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type HashPattern struct {
	Type        string
	Mode        string
	Regex       string
	Description string
}

func getHashPatterns() []HashPattern {
	return []HashPattern{
		{
			Type:        "ntlm",
			Mode:        "1000",
			Regex:       `^[^:]+:[0-9]+:[a-fA-F0-9]{32}:[a-fA-F0-9]{32}:::$`,
			Description: "NTLM (SAM format)",
		},
		{
			Type:        "net-ntlmv2",
			Mode:        "5600",
			Regex:       `^[a-zA-Z0-9]{1,32}:[a-fA-F0-9]{32}:[a-fA-F0-9]{128,}$`,
			Description: "NetNTLMv2",
		},
		{
			Type:        "net-ntlm",
			Mode:        "5500",
			Regex:       `^[a-zA-Z0-9]{1,32}:[a-fA-F0-9]{32}:[a-fA-F0-9]{48,}$`,
			Description: "NetNTLM",
		},
		{
			Type:        "krb5tgs$23",
			Mode:        "13100",
			Regex:       `^\$krb5tgs\$23\$`,
			Description: "Kerberos 5 TGS-REP",
		},
		{
			Type:        "dcc2",
			Mode:        "2100",
			Regex:       `^\$DCC2\$[0-9]+#[^#]+#[a-fA-F0-9]{32}$`,
			Description: "MS Cache v2",
		},
		{
			Type:        "md5",
			Mode:        "0",
			Regex:       `^[a-fA-F0-9]{32}$`,
			Description: "MD5",
		},
		{
			Type:        "sha1",
			Mode:        "100",
			Regex:       `^[a-fA-F0-9]{40}$`,
			Description: "SHA1",
		},
		{
			Type:        "sha256",
			Mode:        "1400",
			Regex:       `^[a-fA-F0-9]{64}$`,
			Description: "SHA2-256",
		},
		{
			Type:        "sha512",
			Mode:        "1700",
			Regex:       `^[a-fA-F0-9]{128}$`,
			Description: "SHA2-512",
		},
		{
			Type:        "mysql-sha1",
			Mode:        "300",
			Regex:       `^\*[a-fA-F0-9]{40}$`,
			Description: "MySQL4.1/MySQL5 SHA1",
		},
	}
}

func DetectHashType(hashFile string) (string, string, error) {
	file, err := os.Open(hashFile)
	if err != nil {
		return "", "", FmtError("error opening hash file:\n%v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", "", FmtError("hash file is empty")
	}
	firstHash := strings.TrimSpace(scanner.Text())

	patterns := getHashPatterns()
	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern.Regex, firstHash)
		if err != nil {
			continue
		}
		if matched {
			LogInfo("Detected hash type: %s (%s)", pattern.Description, pattern.Type)
			return pattern.Type, pattern.Mode, nil
		}
	}

	return "", "", FmtError("unable to detect hash type")
}

func GetHashType(hashType string, hashFile string) string {
	if hashType == "auto" || hashType == "" {
		detectedType, detectedMode, err := DetectHashType(hashFile)
		if err != nil {
			LogCriticalE("Unable to detect hash type: %v", err)
		}
		LogInfo("Detected hash type: %s (%s)", detectedType, detectedMode)
		return detectedMode
	}

	hashTypes := map[string]string{
		"ntlm":       "1000",
		"net-ntlm":   "5500",
		"netntlm":    "5500",
		"net-ntlmv2": "5600",
		"netntlmv2":  "5600",
		"krb5tgs$23": "13100",
		"dcc2":       "2100",
	}

	if val, exists := hashTypes[strings.ToLower(hashType)]; exists {
		LogInfo("Using hash type: %s (%s)", hashType, val)
		return val
	}

	// check if int type:
	hashType = strings.TrimSpace(hashType)
	if _, err := strconv.Atoi(hashType); err == nil {
		LogInfo("Using hash type: %s", hashType)
		return hashType
	}
	LogCriticalE("Invalid hash type: %s", hashType)
	return ""
}
