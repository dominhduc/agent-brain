package knowledge

import (
	"strings"
)

type EntryCluster struct {
	Representative string
	Members        []string
	AvgStrength    float64
	Topic          string
}

const clusterThreshold = 0.35

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
			sim := trigramJaccard(normalizeEntry(entries[i].Message), normalizeEntry(entries[j].Message))
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

		clusters = append(clusters, EntryCluster{
			Representative: entries[indices[0]].Timestamp,
			Members:        members,
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
