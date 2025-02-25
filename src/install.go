package main

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// ****************************************************************************
// GetHashcatExecutable searches for the hashcat executable in predefined paths and returns its location.
// For Windows systems, it looks for hashcat.exe in the current directory and ./hashcat/ subdirectory.
// For non-Windows systems, it looks for hashcat.bin in /opt/hashcat/ and local directories.
// If the path starts with "./", it converts it to an absolute path.
// Returns the path to hashcat executable if found, empty string otherwise.
// Logs a critical error if hashcat is not found in any of the predefined paths.
func GetHashcatExecutable() string {
	paths := []string{}
	if runtime.GOOS == "windows" {
		paths = []string{
			"hashcat.exe",
			"./hashcat/hashcat.exe",
			"./hashcat.exe",
		}
	} else {
		paths = []string{
			"/opt/hashcat/hashcat.bin",
			"./hashcat/hashcat.bin",
			"./hashcat.bin",
		}
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			// if starts with ./ => get absolute path
			// ex: ./hashcat/hashcat.exe => C:\Users\user\hashcat\hashcat.exe
			if strings.HasPrefix(path, "./") {
				if absPath, err := filepath.Abs(path); err == nil {
					return absPath
				} else {
					LogWarning("Error while getting absolute path of %s:\n%v", path, err)
				}
			}
			return path
		}
	}
	LogCritical("Hashcat not found")
	return ""
}

// ****************************************************************************
// InstallHashcat downloads and configures Hashcat password cracking tool and its dependencies.
// It checks if Hashcat is already installed, and if not, proceeds with installation.
// On non-Windows systems, it installs required dependencies (CUDA toolkit and 7zip).
// It creates necessary directories for dictionaries and rules, downloads Hashcat rules,
// and optionally installs dictionary files.
//
// Parameters:
//   - fakeDictInstall: boolean flag to control whether to perform a fake dictionary installation
//
// Returns:
//   - error: nil if successful, error message if installation fails
//
// The function performs the following steps:
//  1. Checks for existing Hashcat installation
//  2. Installs system dependencies (non-Windows only)
//  3. Downloads Hashcat
//  4. Creates required directory structure
//  5. Downloads and installs rules
//  6. Installs dictionary files
func InstallHashcat(fakeDictInstall bool) error {
	if hashcatPath := GetHashcatExecutable(); hashcatPath != "" {
		LogInfo("Hashcat found at %s", hashcatPath)
		return nil
	}

	LogInfo("Hashcat not found, downloading...")

	if runtime.GOOS != "windows" {
		cmd := exec.Command("sh", "-c", "apt update && apt install -y nvidia-cuda-toolkit p7zip-full")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return FmtError("error installing dependencies:\n%v", err)
		}
	}

	if err := DownloadHashcat(); err != nil {
		return FmtError("error downloading hashcat:\n%v", err)
	}

	baseFolder := ""
	if runtime.GOOS == "windows" {
		baseFolder = "."
	} else {
		baseFolder = "/opt"
	}
	os.MkdirAll(baseFolder+"/dico/", 0755)
	os.MkdirAll(baseFolder+"/hashcat/rules/", 0755)
	fp, _ := os.Create(baseFolder + "/hashcat/hashcat.potfile")
	fp.Close()

	//*********************************************************************************************
	// Download all rules:
	InstallRules(baseFolder)

	//*********************************************************************************************
	// Download dico & extract if needed
	InstallDict(baseFolder, fakeDictInstall)
	return nil
}

// ****************************************************************************
// DownloadHashcat downloads and installs the latest version of hashcat from hashcat.net.
// It performs the following steps:
// 1. Downloads the hashcat webpage to find the latest version
// 2. Extracts the download URL using regex
// 3. Downloads the hashcat 7z archive
// 4. Extracts the archive to the appropriate folder based on OS
//   - Windows: current directory (./)
//   - Other OS: /opt/
//
// 5. Renames the extracted folder to just "hashcat"
// 6. Cleans up by removing example files and the downloaded archive
//
// Returns an error if any step fails during the download, extraction or cleanup process.
func DownloadHashcat() error {
	LogInfo("Downloading hashcat metadata for finding version...\n")
	resp, err := http.Get("https://hashcat.net/hashcat/")
	if err != nil {
		return FmtError("error downloading hashcat metadata:\n%v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FmtError("error reading hashcat metadata:\n%v", err)
	}

	re := regexp.MustCompile(`<a href="([^"]+\.7z)"`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return FmtError("error finding hashcat version in metadata")
	}

	LogInfo("Hashcat version found: %s\n", string(matches[1]))

	downloadURL := "https://hashcat.net" + string(matches[1])
	LogInfo("Downloading hashcat from %s\n", downloadURL)
	resp, err = http.Get(downloadURL)
	if err != nil {
		return FmtError("error downloading hashcat:\n%v", err)
	}
	defer resp.Body.Close()
	zippedFile, err := os.CreateTemp("", "hashcat_*.7z")
	if err != nil {
		return FmtError("error creating temp file:\n%v", err)
	}
	_, err = io.Copy(zippedFile, resp.Body)
	if err != nil {
		return FmtError("error saving hashcat:\n%v", err)
	}
	zippedFile.Close()

	outputFolder := ""
	if runtime.GOOS == "windows" {
		outputFolder = "./"
	} else {
		outputFolder = "/opt/"
	}
	if err := SevenZipExtract(zippedFile.Name(), outputFolder); err != nil {
		return FmtError("error extracting hashcat:\n%v", err)
	}
	// sleep 1sec
	time.Sleep(1000 * time.Millisecond)

	// rename folder hashcat-* to hashcat
	if matches, err := filepath.Glob(outputFolder + "hashcat-*"); err != nil {
		return FmtError("error finding hashcat folder:\n%v", err)
	} else {
		if len(matches) > 0 {
			if err := os.Rename(matches[0], outputFolder+"hashcat"); err != nil {
				return FmtError("error renaming hashcat folder:\n%v", err)
			}
		} else {
			return FmtError("7z did not extract hashcat folder")
		}
	}
	// delete example* from hashcat folder
	examples, _ := filepath.Glob(outputFolder + "hashcat/example*")
	for _, example := range examples {
		os.RemoveAll(example)
	}

	// delete zippedFile
	os.Remove(zippedFile.Name())
	return nil
}

// ****************************************************************************
// InstallRules downloads and installs hashcat rules from various GitHub repositories into the specified base folder.
// It handles both direct rule files and gzipped rule files, automatically extracting the latter.
//
// The function downloads popular password cracking rule collections including:
// - OneRuleToRuleThemAll
// - Hob0Rules (hob064 and d3adhob0)
// - clem9669_large
// - SuperUnicorn
// - Pantagrule collections (private v5, hashesorg v6, and royce variants)
//
// Parameters:
//   - baseFolder: Base directory where rules will be installed (rules are placed in baseFolder/hashcat/rules/)
//
// The function logs errors but continues execution if individual rule downloads or extractions fail.
// Gzipped files are automatically extracted after download using 7-Zip.
func InstallRules(baseFolder string) {
	rules := []string{
		"https://github.com/NotSoSecure/password_cracking_rules/raw/master/OneRuleToRuleThemAll.rule",
		"https://github.com/praetorian-inc/Hob0Rules/raw/master/hob064.rule",
		"https://github.com/praetorian-inc/Hob0Rules/raw/master/d3adhob0.rule",
		"https://github.com/clem9669/hashcat-rule/raw/master/clem9669_large.rule",
		"https://github.com/Unic0rn28/hashcat-rules/raw/main/unicorn%20rules/SuperUnicorn.rule",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/private.v5/pantagrule.private.v5.hybrid.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/private.v5/pantagrule.private.v5.one.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/private.v5/pantagrule.private.v5.popular.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/private.v5/pantagrule.private.v5.random.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/hashesorg.v6/pantagrule.hashorg.v6.hybrid.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/hashesorg.v6/pantagrule.hashorg.v6.one.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/hashesorg.v6/pantagrule.hashorg.v6.popular.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/hashesorg.v6/pantagrule.hashorg.v6.random.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/hashesorg.v6/pantagrule.hashorg.v6.raw1m.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/private.hashorg.royce/pantagrule.hybrid.royce.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/private.hashorg.royce/pantagrule.one.royce.rule.gz",
		"https://github.com/rarecoil/pantagrule/raw/refs/heads/master/rules/private.hashorg.royce/pantagrule.random.royce.rule.gz",
	}
	for _, rule := range rules {
		outFile := baseFolder + "/hashcat/rules/" + filepath.Base(rule)
		if err := DownloadHTTPFile(rule, outFile); err != nil {
			LogError("Error while downloading rule %s:\n%v", rule, err)
			continue
		}
		if strings.HasSuffix(outFile, ".gz") {
			if err := SevenZipExtract(outFile, baseFolder+"/hashcat/rules/"); err != nil {
				LogError("Error while extracting rule %s:\n%v", rule, err)
				continue
			}
		}
	}
}

// ****************************************************************************
// InstallDict downloads and processes dictionary files for password cracking.
//
// The function downloads various wordlists from different sources including weakpass.com,
// SecLists repository, and custom wordlists. It handles both regular text files and
// compressed archives (.7z).
//
// Parameters:
//   - baseFolder: The base directory where dictionary files will be stored
//   - fakeDictInstall: If true, only creates empty placeholder files instead of downloading
//
// The function performs the following operations:
//  1. Downloads dictionary files from provided URLs
//  2. Extracts .7z archives if necessary
//  3. Renames .txt files to .dico extension
//  4. Handles special case for split files (piotrcki wordlist) by merging parts
//
// Any errors during download, extraction, or file operations are logged but don't stop
// the overall process - the function continues with the next file.
func InstallDict(baseFolder string, fakeDictInstall bool) {
	dico := []string{
		"https://download.weakpass.com/wordlists/2012/weakpass_4.txt.7z",
		"https://weakpass.com/download/1931/Hashes.org.7z",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/dutch_common_wordlist.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/dutch_passwordlist.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/dutch_wordlist",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/german_misc.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/months.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/richelieu-french-top5000.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/days.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/darkweb2017-top10000.txt",
		"https://github.com/danielmiessler/SecLists/blob/master/Passwords/Leaked-Databases/porn-unknown.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Leaked-Databases/youporn2012.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Leaked-Databases/fortinet-2021.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Leaked-Databases/NordVPN.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Leaked-Databases/Ashley-Madison.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Leaked-Databases/alleged-gmail-passwords.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Pwdb-Public/Wordlists/ignis-1M.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Pwdb-Public/Wordlists/Language-Specifics/ignis-french-150.txt",
		"https://github.com/danielmiessler/SecLists/raw/refs/heads/master/Passwords/Pwdb-Public/Wordlists/Language-Specifics/ignis-german-150.txt",
	}
	for _, dico := range dico {
		outFile := baseFolder + "/dico/" + filepath.Base(dico)
		if fakeDictInstall {
			// touch file
			fp, _ := os.Create(outFile + ".dico")
			fp.Close()
			LogInfo("[FAKE] Downloading dico %s\n", dico)
			continue
		}

		LogInfo("Downloading dico %s\n", dico)
		if err := DownloadHTTPFile(dico, outFile); err != nil {
			LogError("Error while downloading dico %s:\n%v", dico, err)
			continue
		}
		if strings.HasSuffix(outFile, ".7z") {
			if err := SevenZipExtract(outFile, baseFolder+"/dico/"); err != nil {
				LogError("Error while extracting dico %s:\n%v", dico, err)
				continue
			}
		}
		// mv .txt to .dico
		if !strings.HasSuffix(outFile, ".dico") {
			if err := os.Rename(outFile, strings.Replace(outFile, ".txt", ".dico", 1)); err != nil {
				LogWarning("Error while renaming dico %s:\n%v", dico, err)
			}
		}
	}

	/*
		wget https://github.com/piotrcki/wordlist/releases/download/v0.0.0/piotrcki-wordlist.txt.xz.part00 -O /opt/hashcat/dico/piotrcki-wordlist.txt.xz.part00
		wget https://github.com/piotrcki/wordlist/releases/download/v0.0.0/piotrcki-wordlist.txt.xz.part01 -O /opt/hashcat/dico/piotrcki-wordlist.txt.xz.part01
		cat /opt/hashcat/dico/piotrcki-wordlist.txt.xz.part00 /opt/hashcat/dico/piotrcki-wordlist.txt.xz.part01 > piotrcki-wordlist.txt.xz
		rm /opt/hashcat/dico/piotrcki-wordlist.txt.xz.part00 /opt/hashcat/dico/piotrcki-wordlist.txt.xz.part01
	*/
	dico = []string{
		"https://github.com/piotrcki/wordlist/releases/download/v0.0.0/piotrcki-wordlist.txt.xz.part00",
		"https://github.com/piotrcki/wordlist/releases/download/v0.0.0/piotrcki-wordlist.txt.xz.part01",
	}
	for _, dico := range dico {
		outFile := baseFolder + "/dico/" + filepath.Base(dico)
		if fakeDictInstall {
			// touch file
			fp, _ := os.Create(outFile + ".dico")
			fp.Close()
			LogInfo("[FAKE] Downloading dico %s\n", dico)
			continue
		}

		LogInfo("Downloading dico %s\n", dico)
		if err := DownloadHTTPFile(dico, outFile); err != nil {
			LogError("Error while downloading dico %s:\n%v", dico, err)
			continue
		}
	}
	// merge two files
	if !fakeDictInstall {
		// Append part01 to part00
		part00 := baseFolder + "/dico/piotrcki-wordlist.txt.xz.part00"
		part01 := baseFolder + "/dico/piotrcki-wordlist.txt.xz.part01"

		// Open part01 for reading
		inFile, err := os.Open(part01)
		if err != nil {
			LogError("Error opening part01 file:\n%v", err)
			return
		}
		defer inFile.Close()

		// Open part00 for appending
		outFile, err := os.OpenFile(part00, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			LogError("Error opening part00 file for appending:\n%v", err)
			return
		}
		defer outFile.Close()

		// Append part01 to part00
		if _, err := io.Copy(outFile, inFile); err != nil {
			LogError("Error appending files:\n%v", err)
			return
		}

		// Rename part00 to final name and cleanup
		if err := os.Rename(part00, baseFolder+"/dico/piotrcki-wordlist.txt.xz"); err != nil {
			LogError("Error renaming merged file:\n%v", err)
			return
		}
		os.Remove(part01)
		outFile.Close()
		if err := SevenZipExtract(part00, baseFolder+"/dico/"); err != nil {
			LogError("Error while extracting dico %s:\n%v", dico, err)
		} else {
			// renmae .txt to .dico
			if err := os.Rename(part00, strings.Replace(part00, ".txt", ".dico", 1)); err != nil {
				LogError("Error while renaming dico %s:\n%v", dico, err)
			}
		}
	}
}
