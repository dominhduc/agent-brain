package knowledge

import (
	"strings"
)

type EntryCluster struct {
	Representative string
	MemberIndices  []int
	Members        []string
	AvgStrength    float64
	Topic          string
}

const clusterThreshold = 0.40

func ClusterEntries(entries []TopicEntry, topic string) []EntryCluster {
	if len(entries) == 0 {
		return nil
	}

	n := len(entries)
	parent := make([]int, n)
	rank := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y int) {
		px, py := find(x), find(y)
		if px == py {
			return
		}
		if rank[px] < rank[py] {
			px, py = py, px
		}
		parent[py] = px
		if rank[px] == rank[py] {
			rank[px]++
		}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			sim := trigramJaccard(stripStopWords(normalizeEntry(entries[i].Message)), stripStopWords(normalizeEntry(entries[j].Message)))
			if sim >= clusterThreshold {
				union(i, j)
			}
		}
	}

	sets := make(map[int][]int)
	for i := 0; i < n; i++ {
		root := find(i)
		sets[root] = append(sets[root], i)
	}

	var clusters []EntryCluster
	for _, indices := range sets {
		if len(indices) < 2 {
			continue
		}

		var members []string
		for _, idx := range indices {
			members = append(members, entries[idx].Timestamp)
		}

		var totalStrength float64
		var strengthCount int
		for _, idx := range indices {
			if len(entries) > idx {
				s := entryStrength(entries[idx].Message)
				totalStrength += s
				strengthCount++
			}
		}
		avgStr := 0.0
		if strengthCount > 0 {
			avgStr = totalStrength / float64(strengthCount)
		}

		clusters = append(clusters, EntryCluster{
			Representative: entries[indices[0]].Timestamp,
			MemberIndices:  indices,
			Members:        members,
			AvgStrength:    avgStr,
			Topic:          topic,
		})
	}

	return clusters
}

func ClusterEntriesForTopic(topic, brainDir string) ([]EntryCluster, error) {
	entries, err := GetTopicEntriesForDir(topic, brainDir)
	if err != nil {
		return nil, err
	}
	return ClusterEntries(entries, topic), nil
}

func ClusterAllTopics(brainDir string) ([]EntryCluster, error) {
	var allClusters []EntryCluster
	for _, topic := range AvailableTopics() {
		clusters, err := ClusterEntriesForTopic(topic, brainDir)
		if err != nil {
			continue
		}
		allClusters = append(allClusters, clusters...)
	}
	return allClusters, nil
}

func normalizeTextForCluster(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}

func entryStrength(message string) float64 {
	return 1.0
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
	"be": true, "has": true, "have": true, "had": true, "not": true, "this": true,
	"that": true, "it": true, "its": true, "may": true, "can": true, "will": true,
	"should": true, "when": true, "use": true, "using": true, "used": true,
	"must": true, "also": true, "no": true, "via": true, "get": true,
}

func stripStopWords(text string) string {
	words := strings.Fields(text)
	var filtered []string
	for _, w := range words {
		if !stopWords[w] {
			filtered = append(filtered, w)
		}
	}
	return strings.Join(filtered, " ")
}
