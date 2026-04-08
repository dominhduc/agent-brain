package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/index"
	"github.com/dominhduc/agent-brain/internal/secrets"
	"github.com/dominhduc/agent-brain/internal/wm"
)

var knownTopics = []string{"ui", "backend", "infrastructure", "database", "security", "testing", "architecture", "general"}

func isKnownTopic(arg string) bool {
	for _, t := range knownTopics {
		if strings.EqualFold(t, arg) {
			return true
		}
	}
	return false
}

func cmdAdd() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: brain add <topic> \"<message>\"")
		fmt.Println("       brain add <8-topic> <topic> \"<message>\"")
		fmt.Println("       brain add --wm \"<message>\"")
		fmt.Println("Topics: gotcha, pattern, decision, architecture, memory")
		fmt.Println("8-Topics: ui, backend, infrastructure, database, security, testing, architecture, general")
		fmt.Println("What to do: provide a topic and a message to add.")
		os.Exit(1)
	}

	wmFlag := hasFlag("--wm")
	if wmFlag {
		if len(os.Args) < 4 {
			fmt.Println("Usage: brain add --wm \"<message>\"")
			os.Exit(1)
		}
		message := strings.Join(os.Args[3:], " ")
		brainDir, err := brain.FindBrainDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := wm.Push(brainDir, message, 0.5); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Added to working memory.")
		return
	}

	var entryTopic string
	var messageTopic string
	var message string

	if len(os.Args) >= 5 && isKnownTopic(os.Args[2]) {
		topicTag := os.Args[2]
		entryTopic = os.Args[3]
		message = strings.Join(os.Args[4:], " ")
		messageTopic = topicTag
	} else {
		entryTopic = os.Args[2]
		message = strings.Join(os.Args[3:], " ")
	}

	if len(message) > maxMessageLen {
		fmt.Fprintf(os.Stderr, "Error: message too long (%d bytes, max %d).\nWhat to do: shorten your message or split it into multiple entries.\n", len(message), maxMessageLen)
		os.Exit(1)
	}

	if secrets.HasSecrets(message) {
		findings := secrets.Scan(message)
		fmt.Fprintf(os.Stderr, "Error: your message may contain a secret (detected: %s).\n", findings[0].Type)
		fmt.Fprintln(os.Stderr, "What to do: redact the sensitive value and try again.")
		os.Exit(1)
	}

	topicMap := map[string]string{
		"gotcha":       "gotchas",
		"pattern":      "patterns",
		"decision":     "decisions",
		"architecture": "architecture",
		"memory":       "memory",
	}

	normalized, ok := topicMap[strings.ToLower(entryTopic)]
	if !ok {
		fmt.Printf("Unknown topic '%s'. Available topics: gotcha, pattern, decision, architecture, memory\n", entryTopic)
		fmt.Println("What to do: use one of the listed topic names.")
		os.Exit(1)
	}

	if err := brain.AddEntry(normalized, message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: make sure you are in a project with .brain/ initialized.\n", err)
		os.Exit(1)
	}

	if messageTopic != "" {
		brainDir, err := brain.FindBrainDir()
		if err == nil {
			idx, err := index.Load(brainDir)
			if err == nil {
				timestamp := index.MakeKey(normalized, "")
				_ = timestamp
				for k := range idx.Entries {
					if strings.HasPrefix(k, normalized+":") {
						parts := strings.SplitN(k, ":", 2)
						if len(parts) == 2 && parts[1] != "" {
							entry, found := idx.Get(normalized, parts[1])
							if found {
								hasTopic := false
								for _, t := range entry.Topics {
									if t == messageTopic {
										hasTopic = true
										break
									}
								}
								if !hasTopic {
									entry.Topics = append(entry.Topics, messageTopic)
									idx.Set(normalized, parts[1], entry)
								}
								break
							}
						}
					}
				}
				idx.Save(brainDir)
			}
		}
	}

	fmt.Printf("Added to %s\n", normalized)
	if messageTopic != "" {
		fmt.Printf("Tagged with topic: %s\n", messageTopic)
	}
}
