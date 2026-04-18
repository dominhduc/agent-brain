package embed

import (
	"sort"
)

type HybridResult struct {
	Key         string
	Topic       string
	Message     string
	KeywordRank int
	VectorRank  int
	RRFCombined float64
}

const rrfK = 60

func RRFScore(keywordResults []string, vectorResults []SearchResult, key string) float64 {
	var score float64

	for i, k := range keywordResults {
		if k == key {
			score += 1.0 / float64(rrfK+i+1)
			break
		}
	}

	for i, v := range vectorResults {
		if v.Key == key {
			score += 1.0 / float64(rrfK+i+1)
			break
		}
	}

	return score
}

func HybridSearch(query string, keywordResults []string, vectorResults []SearchResult, topK int) []HybridResult {
	allKeys := make(map[string]bool)
	for _, k := range keywordResults {
		allKeys[k] = true
	}
	for _, v := range vectorResults {
		allKeys[v.Key] = true
	}

	var candidates []HybridResult
	for key := range allKeys {
		var keywordRank, vectorRank int = -1, -1
		for i, k := range keywordResults {
			if k == key {
				keywordRank = i
				break
			}
		}
		for i, v := range vectorResults {
			if v.Key == key {
				vectorRank = i
				break
			}
		}

		candidates = append(candidates, HybridResult{
			Key:         key,
			KeywordRank: keywordRank,
			VectorRank:  vectorRank,
			RRFCombined: RRFScore(keywordResults, vectorResults, key),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].RRFCombined > candidates[j].RRFCombined
	})

	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	return candidates
}
