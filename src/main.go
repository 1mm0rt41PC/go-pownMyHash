package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	HashcatBin         string
	HashcatDir         string
	HashcatPotfile     string
	HashcatFound       uint64
	HashcatPotfileSize int64
	RulesDir           string
	Rules              []string
	DictsDir           string
	HistoricalDict     string
	StatsFile          string
	CustomDict         string
	HashType           string
	HashFile           string
	Phase              string
	DictRanking        *DictRanking
}

func InvokeHashcatKnowledgeLoop(cfg *Config, dict string) {
	LogInfo("*****************************************************************************************")
	LogInfo("Starting %s phase ...", dict)
	cfg.Phase = fmt.Sprintf("Knowledge phase %s", filepath.Base(dict))
	initialCount := exportFoundPasswords(cfg)
	if initialCount == 0 {
		LogInfo("No passwords found, skipping Custom phase")
		return
	}

	for _, rule := range cfg.Rules {
		LogInfo("Applying rule: %s with %s", filepath.Base(rule), dict)
		if err := InvokeHashcatDictWithRule(cfg, dict, rule); err != nil {
			LogError("Error running hashcat with rule %s:\n%v", rule, err)
			continue
		}
	}
	newCount := exportFoundPasswords(cfg)
	if newCount > initialCount {
		LogInfo("Found new passwords! Recursively trying rules again...")
		initialCount = newCount
		InvokeHashcatKnowledgeLoop(cfg, dict)
	}
}

func startHashcatAutomation(cfg *Config) {
	LogInfo("\n\033[42m***************************************************************************************************\nChoose the order of attack:\033[0m")
	if cfg.HashType == "1000" {
		fmt.Println("LM. LM Attack first")
	}
	fmt.Println("1. Historical dictionary")
	fmt.Println("2. Custom dictionary")
	fmt.Println("3. Dictionaries")
	fmt.Println("4. Brute force password with len=8 with automask")
	fmt.Println("5. Brute force password with len=8")
	fmt.Println("6. Rules stacking with best64 rule on Historical Dict")
	fmt.Println("7. Rules stacking with best64 rule on all dico")
	if cfg.HashType == "1000" {
		fmt.Printf("Enter your choice (default: LM,1,2,3,4,5,6,7): ")
	} else {
		fmt.Printf("Enter your choice (default: 1,2,3,4,5,6,7): ")
	}
	os.Stdout.Sync()
	var userOrder string
	fmt.Scanln(&userOrder)
	if userOrder == "" {
		if cfg.HashType == "1000" {
			userOrder = "LM,1,2,3,4,5,6,7"
		} else {
			userOrder = "1,2,3,4,5,6,7"
		}
	}
	orders := strings.Split(userOrder, ",")
	for _, order := range orders {
		switch order {
		case "LM":
			if AskForConfirmation("Audit LM hashes first?") {
				cfg.HashType = "3000"
				cfg.Phase = "Audit LM hashes"
				LogInfo("Running hashcat with LM hashes audit")
				if err := InvokeHashcatBruteForce(cfg, "?a?a?a?a?a?a", []string{"-i"}); err != nil {
					LogCritical("Error running hashcat with LM hashes:\n%v", err)
				}
				if err := InvokeHashcatBruteForce(cfg, "?a?a?a?a?a?a?a", nil); err != nil {
					LogCritical("Error running hashcat with LM hashes:\n%v", err)
				}
				cfg.HashType = "1000"
			}
		case "1":
			if AskForConfirmation("Start hashcat on Historical dictionary?") {
				InvokeHashcatKnowledgeLoop(cfg, cfg.HistoricalDict)
			}
		case "2":
			if AskForConfirmation("Start hashcat on Custom dictionary?") {
				InvokeHashcatKnowledgeLoop(cfg, cfg.CustomDict)
			}
		case "3":
			dicts, err := filepath.Glob(filepath.Join(cfg.DictsDir, "*.dico"))
			if err != nil {
				LogCritical("Error reading dictionaries directory (%s):\n%v", cfg.DictsDir, err)
				return
			}
			if len(dicts) == 0 {
				LogCritical("No dictionaries found in %s", cfg.DictsDir)
				return
			}
			if AskForConfirmation(fmt.Sprintf("Start hashcat on %d dictionaries?", len(dicts))) {
				cfg.DictRanking = NewDictRanking(cfg)
				initialCount := uint64(0)
				newCount := uint64(0)
				for i, dict := range cfg.DictRanking.RankDictionaries(dicts) {
					if !AskForConfirmation(fmt.Sprintf("Start hashcat on dictionary %s ?", dict)) {
						continue
					}
					cfg.Phase = fmt.Sprintf("Dictionnary phase %d/%d %s", i, len(dicts), filepath.Base(dict))
					initialCount = exportFoundPasswords(cfg)
					LogInfo("Running hashcat with dictionary (%s) and no rule", dict)
					if err := InvokeHashcatDictWithRule(cfg, dict, ""); err != nil {
						LogCritical("Error running hashcat with dictionary %s:\n%v", dict, err)
						continue
					}
					// Update stats
					newCount = exportFoundPasswords(cfg)
					cfg.DictRanking.UpdateStats(dict, newCount-initialCount)
					initialCount = newCount
					// Apply rules
					for j, rule := range cfg.Rules {
						cfg.Phase = fmt.Sprintf("Dictionnary phase %d/%d %s, width rule %d/%d %s", i, len(dicts), filepath.Base(dict), j, len(cfg.Rules), filepath.Base(rule))
						LogInfo("Running hashcat with dictionary (%s) and rule (%s)", cfg.HistoricalDict, rule)
						InvokeHashcatDictWithRule(cfg, dict, rule)
						// Update stats
						newCount = exportFoundPasswords(cfg)
						cfg.DictRanking.UpdateStats(dict, newCount-initialCount)
						initialCount = newCount
					}
				}
				InvokeHashcatKnowledgeLoop(cfg, cfg.CustomDict)
			}
		case "4":
			if AskForConfirmation("Brute force password with len=8 with automask") {
				LogInfo("Running hashcat with automask")
				cfg.Phase = fmt.Sprintf("Brute force password with len=8 with automask")
				if err := InvokeHashcat(cfg, []string{"-a", "3", "--increment", "--increment-min", "8", "--increment-max", "10"}); err != nil {
					LogCritical("Error running hashcat with automask:\n%v", err)
				}
			}

		case "5":
			if AskForConfirmation("Brute force password with len=8") {
				LogInfo("Running hashcat with len=8")
				cfg.Phase = fmt.Sprintf("Brute force password with len=8")
				if err := InvokeHashcatBruteForce(cfg, "?a?a?a?a?a?a?a?a", nil); err != nil {
					LogCritical("Error running hashcat with len=8:\n%v", err)
				}
			}
		case "6":
			/*
				if title "Using potfile as dico with all rules with stacking with best64 rule "; then
					hashcat 0 `absPath $FINDINGS`
					for rule in $(find $HC/rules/ -type f);do
						title "Using potfile as dico with rule $rule" 0
						hashcat 0 `absPath $FINDINGS` -r `absPath $rule` -r `absPath $HC/rules/best64.rule` --loopback
					done
				fi
			*/
			if AskForConfirmation("Rules stacking with best64 rule on Historical Dict") {
				cfg.Phase = fmt.Sprintf("Rules stacking with best64 rule on Historical Dict")
				best64Rule := filepath.Join(cfg.RulesDir, "best64.rule")
				if _, err := os.Stat(best64Rule); err != nil {
					LogCritical("best64.rule not found in %s", cfg.RulesDir)
					continue
				}
				for _, rule := range cfg.Rules {
					LogInfo("Using potfile as dico with rule %s", rule)
					if err := InvokeHashcatStackingRules(cfg, []string{cfg.HistoricalDict, "-r", rule, "-r", best64Rule, "--loopback"}); err != nil {
						LogCritical("Error running hashcat with rule %s:\n%v", rule, err)
						continue
					}
				}
			}

		case "7":
			if AskForConfirmation("Rules stacking with best64 rule on all dico") {
				best64Rule := filepath.Join(cfg.RulesDir, "best64.rule")
				if _, err := os.Stat(best64Rule); err != nil {
					LogCritical("best64.rule not found in %s", cfg.RulesDir)
					continue
				}
				dicts, err := filepath.Glob(filepath.Join(cfg.DictsDir, "*.dico"))
				if err != nil {
					LogCritical("Error reading dictionaries directory (%s):\n%v", cfg.DictsDir, err)
					return
				}
				if len(dicts) == 0 {
					LogCritical("No dictionaries found in %s", cfg.DictsDir)
					return
				}
				for _, dict := range dicts {
					for _, rule := range cfg.Rules {
						LogInfo("Using potfile as dico with rule %s", rule)
						cfg.Phase = fmt.Sprintf("Rules stacking with best64 rule on %s dico with rule %s", filepath.Base(dict), filepath.Base(rule))
						if err := InvokeHashcatStackingRules(cfg, []string{dict, "-r", rule, "-r", best64Rule, "--loopback"}); err != nil {
							LogCritical("Error running hashcat with rule %s:\n%v", rule, err)
							continue
						}
					}
				}
			}
		}
	}
}

func main() {
	hashType := flag.String("type", "auto", "Hash type (auto, ntlm, net-ntlm, net-ntlmv2, krb5tgs$23, dcc2, or numeric mode)")
	hashFile := flag.String("hashes", "", "Path to the hash file")
	rulesDir := flag.String("rules", "", "Path to rules directory")
	dictsDir := flag.String("dicts", "dico", "Path to dictionaries directory")
	historicalDict := flag.String("historical", "pownMyHash.dico", "Path to historical dictionary")
	dictStatsFile := flag.String("dict-stats", "dict-stats.json", "Path to dictionary statistics file")
	fakeFlag := flag.Bool("fake", false, "Fake flag to bypass hashcat installation")

	flag.Parse()

	if *hashFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := InstallHashcat(*fakeFlag); err != nil {
		LogCriticalE("Unable to install hashcat: %v", err)
	}

	// check if hashes file exists
	if _, err := os.Stat(*hashFile); err != nil {
		LogCriticalE("Hash file not found: %v", err)
	}

	cfg := &Config{
		HashcatBin:     GetHashcatExecutable(),
		RulesDir:       *rulesDir,
		DictsDir:       *dictsDir,
		HistoricalDict: *historicalDict,
		HashType:       *hashType,
		HashFile:       *hashFile,
		StatsFile:      *dictStatsFile,
	}

	// Convert hashes file relative path to absolute path
	if absPath, err := filepath.Abs(cfg.HashFile); err != nil {
		LogCriticalE("Error while getting absolute path of %s:\n%v", *hashFile, err)
	} else {
		cfg.HashFile = absPath
	}
	LogInfo("Hash file: %s", cfg.HashFile)
	cfg.HashType = GetHashType(cfg.HashType, cfg.HashFile)
	cfg.CustomDict = cfg.HashFile + ".dict"
	LogInfo("Custom dictionary: %s", cfg.CustomDict)

	cfg.HashcatDir = filepath.Dir(cfg.HashcatBin)
	cfg.HashcatPotfile = filepath.Join(cfg.HashcatDir, "hashcat.potfile")
	if cfg.RulesDir == "" {
		cfg.RulesDir = filepath.Join(cfg.HashcatDir, "rules")
	}
	LogInfo("Rules directory: %s", cfg.RulesDir)
	rules, err := filepath.Glob(filepath.Join(cfg.RulesDir, "*.rule"))
	if err != nil {
		LogCriticalE("Error reading rules directory (%s): %v\n", cfg.RulesDir, err)
	}
	cfg.Rules = rules
	LogInfo("Rules: %v", cfg.Rules)

	// Adjust historicalDict to be in the same folder as the current executable
	if !filepath.IsAbs(cfg.HistoricalDict) {
		// Check if file exists in cwd, if it exist => convert to abs
		if _, err := os.Stat(cfg.HistoricalDict); err == nil {
			absPath, err := filepath.Abs(cfg.HistoricalDict)
			if err != nil {
				LogCriticalE("Error while getting absolute path of %s:\n%v", cfg.HistoricalDict, err)
			}
			cfg.HistoricalDict = absPath
			LogInfo("Historical dictionary found: %s", cfg.HistoricalDict)
		} else {
			ced := filepath.Dir(os.Args[0])
			absPath, err := filepath.Abs(ced)
			if err != nil {
				LogCriticalE("Error while getting absolute path of %s:\n%v", ced, err)
			}
			cfg.HistoricalDict = filepath.Join(absPath, cfg.HistoricalDict)
			// If not exist ! show Warning
			if _, err := os.Stat(cfg.HistoricalDict); err != nil {
				LogWarning("Historical dictionary (%s) not found:\n%v", cfg.HistoricalDict, err)
			} else {
				LogInfo("Historical dictionary found: %s", cfg.HistoricalDict)
			}
		}
	}

	if !filepath.IsAbs(cfg.HistoricalDict) {
		// Check if file exists in cwd, if it exist => convert to abs
		if _, err := os.Stat(cfg.HistoricalDict); err == nil {
			absPath, err := filepath.Abs(cfg.HistoricalDict)
			if err != nil {
				LogCriticalE("Error while getting absolute path of %s:\n%v", cfg.HistoricalDict, err)
			}
			cfg.HistoricalDict = absPath
			LogInfo("Historical dictionary found: %s", cfg.HistoricalDict)
		} else {
			ced := filepath.Dir(os.Args[0])
			absPath, err := filepath.Abs(ced)
			if err != nil {
				LogCriticalE("Error while getting absolute path of %s:\n%v", ced, err)
			}
			cfg.HistoricalDict = filepath.Join(absPath, cfg.HistoricalDict)
			// If not exist ! show Warning
			if _, err := os.Stat(cfg.HistoricalDict); err != nil {
				LogWarning("Historical dictionary (%s) not found:\n%v", cfg.HistoricalDict, err)
			} else {
				LogInfo("Historical dictionary found: %s", cfg.HistoricalDict)
			}
		}
	}

	if !filepath.IsAbs(cfg.StatsFile) {
		// Check if file exists in cwd, if it exist => convert to abs
		if _, err := os.Stat(cfg.StatsFile); err == nil {
			absPath, err := filepath.Abs(cfg.StatsFile)
			if err != nil {
				LogCriticalE("Error while getting absolute path of %s:\n%v", cfg.StatsFile, err)
			}
			cfg.StatsFile = absPath
			LogInfo("Dictionaries stats found: %s", cfg.StatsFile)
		} else {
			ced := filepath.Dir(os.Args[0])
			absPath, err := filepath.Abs(ced)
			if err != nil {
				LogCriticalE("Error while getting absolute path of %s:\n%v", ced, err)
			}
			cfg.StatsFile = filepath.Join(absPath, cfg.StatsFile)
			// If not exist ! show Warning
			if _, err := os.Stat(cfg.StatsFile); err != nil {
				LogWarning("Dictionaries stats (%s) not found:\n%v", cfg.StatsFile, err)
			} else {
				LogInfo("Dictionaries stats found: %s", cfg.StatsFile)
			}
		}
	}

	if !filepath.IsAbs(cfg.DictsDir) {
		// Check if /opt/{cfg.DictsDir} exists
		if _, err := os.Stat(filepath.Join("/opt", cfg.DictsDir)); err == nil {
			cfg.DictsDir = filepath.Join("/opt", cfg.DictsDir)
			LogInfo("Dictionary path found: %s", cfg.DictsDir)
		} else {
			// Check if file exists in cwd, if it exist => convert to abs
			if _, err := os.Stat(cfg.DictsDir); err == nil {
				absPath, err := filepath.Abs(cfg.DictsDir)
				if err != nil {
					LogCriticalE("Error while getting absolute path of %s:\n%v", cfg.DictsDir, err)
				}
				cfg.DictsDir = absPath
				LogInfo("Dictionary path found: %s", cfg.DictsDir)
			} else {
				ced := filepath.Dir(os.Args[0])
				absPath, err := filepath.Abs(ced)
				if err != nil {
					LogCriticalE("Error while getting absolute path of %s:\n%v", ced, err)
				}
				cfg.DictsDir = filepath.Join(absPath, cfg.DictsDir)
				// If not exist ! show Warning
				if _, err := os.Stat(cfg.DictsDir); err != nil {
					LogWarning("Dictionary path (%s) not found:\n%v", cfg.DictsDir, err)
				} else {
					LogInfo("Dictionary path found: %s", cfg.DictsDir)
				}
			}
		}
	}

	startHashcatAutomation(cfg)
}
