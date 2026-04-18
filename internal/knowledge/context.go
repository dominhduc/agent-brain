package knowledge

import (
	"os/exec"
	"strings"
)

func DetectWorkContext(cwd string) ([]string, error) {
	cmd := exec.Command("git", "-C", cwd, "diff", "--stat", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			files = append(files, parts[0])
		}
	}

	topicSet := make(map[string]bool)
	for _, file := range files {
		fileTopics := DetectTopics(file)
		for _, t := range fileTopics {
			topicSet[t] = true
		}
	}

	var topics []string
	for t := range topicSet {
		topics = append(topics, t)
	}
	return topics, nil
}
