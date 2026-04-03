package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dominhduc/agent-brain/internal/brain"
)

func cmdGet(jsonFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain get <topic>")
		fmt.Println("Topics: memory, gotchas, patterns, decisions, architecture, all")
		fmt.Println("What to do: specify a topic name to retrieve.")
		os.Exit(1)
	}

	topic := os.Args[2]

	if topic == "all" {
		if jsonFlag {
			topics := map[string]string{}
			for _, t := range brain.AvailableTopics() {
				c, err := brain.GetTopic(t)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", t, err)
					os.Exit(1)
				}
				topics[t] = c
			}
			data, _ := json.MarshalIndent(topics, "", "  ")
			fmt.Println(string(data))
		} else {
			content, err := brain.GetAllTopics()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
				os.Exit(1)
			}
			fmt.Println(content)
		}
		return
	}

	content, err := brain.GetTopic(topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if jsonFlag {
		data, _ := json.MarshalIndent(map[string]string{topic: content}, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println(content)
	}
}
