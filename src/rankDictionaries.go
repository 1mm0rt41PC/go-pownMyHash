package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

type kv struct {
	Key   string
	Value uint64
}

func (dr *DictRanking) SortDict() []kv {
	var ss []kv
	for k, v := range dr.Rank {
		ss = append(ss, kv{dr.DictsDir + k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	return ss
}

// DictRanking structure pour gérer le classement des dictionnaires
type DictRanking struct {
	StatsFile string
	DictsDir  string
	Rank      map[string]uint64
}

func NewDictRanking(cfg *Config) *DictRanking {
	dr := &DictRanking{
		StatsFile: cfg.StatsFile,
		DictsDir:  cfg.DictsDir,
		Rank:      make(map[string]uint64),
	}

	// Charger les statistiques existantes si le fichier existe
	if _, err := os.Stat(dr.StatsFile); err == nil {
		data, err := os.ReadFile(dr.StatsFile)
		if err != nil {
			LogWarning("Error reading dict stats file:\n%v", err)
			return dr
		}
		if err := json.Unmarshal(data, &dr.Rank); err != nil {
			LogWarning("Error parsing dict stats file, starting with empty stats: %v", err)
		}
	}

	return dr
}

// UpdateStats met à jour les statistiques pour un dictionnaire
func (dr *DictRanking) UpdateStats(dictPath string, foundCount uint64) {
	dictPath = filepath.Base(dictPath)
	LogInfo("Found %d new passwords via dict %s", foundCount, dictPath)
	stats, exists := dr.Rank[dictPath]
	if !exists {
		dr.Rank[dictPath] = foundCount
		dr.Save()
		return
	}

	dr.Rank[dictPath] = stats + foundCount
	dr.Save()
}

// RankDictionaries renvoie la liste des dictionnaires triés par efficacité
func (dr *DictRanking) RankDictionaries(dictPaths []string) []string {
	// Créer une entrée pour chaque dictionnaire découvert s'il n'existe pas déjà
	for _, path := range dictPaths {
		path = filepath.Base(path)
		if _, exists := dr.Rank[path]; !exists {
			dr.Rank[path] = 0
		}
	}

	ss := dr.SortDict()

	// Convertir les stats triées en chemins de dictionnaires
	rankedPaths := make([]string, len(ss))
	for i, stats := range ss {
		rankedPaths[i] = stats.Key
	}
	return rankedPaths
}

// Save sauvegarde les statistiques dans un fichier
func (dr *DictRanking) Save() error {
	data, err := json.MarshalIndent(dr.Rank, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dr.StatsFile, data, 0644)
}

// PrintRanking affiche le classement des dictionnaires
func (dr *DictRanking) PrintRanking(dictPaths []string) {
	rankedDicts := dr.RankDictionaries(dictPaths)

	LogInfo("Dictionary Ranking:")
	for i, dict := range rankedDicts {
		stats := dr.Rank[dict]
		LogInfo("%2d. | %05d | %s", i+1, dict, stats)
	}
}
