package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

var topicFileNames = []string{"gotchas.md", "patterns.md", "decisions.md", "architecture.md"}

func cmdSync() {
	importFlag := hasFlag("--import")
	dryRunFlag := hasFlag("--dry-run")

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no .brain/ directory found.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain init' first.")
		os.Exit(1)
	}

	projectDir := filepath.Dir(brainDir)
	docsBrainDir := filepath.Join(projectDir, "docs", "brain")

	if importFlag {
		syncImport(docsBrainDir, brainDir, dryRunFlag)
	} else {
		syncExport(brainDir, docsBrainDir, dryRunFlag)
	}
}

func syncExport(brainDir, docsBrainDir string, dryRun bool) {
	exported := 0
	skipped := 0

	for _, name := range topicFileNames {
		src := filepath.Join(brainDir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			skipped++
			continue
		}

		if isStubContent(name, data) {
			skipped++
			continue
		}

		if dryRun {
			fmt.Printf("  Would export: %s (%d bytes)\n", name, len(data))
			exported++
			continue
		}

		dst := filepath.Join(docsBrainDir, name)
		if err := os.MkdirAll(docsBrainDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating docs/brain/: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", dst, err)
			continue
		}
		fmt.Printf("  Exported: %s\n", name)
		exported++
	}

	if dryRun {
		fmt.Printf("\nDry run: %d files would be exported, %d skipped (missing or stub).\n", exported, skipped)
	} else {
		fmt.Printf("\nExported %d topic files to docs/brain/ (%d skipped).\n", exported, skipped)
	}
}

func syncImport(docsBrainDir, brainDir string, dryRun bool) {
	imported := 0
	skipped := 0

	for _, name := range topicFileNames {
		src := filepath.Join(docsBrainDir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			skipped++
			continue
		}

		if isStubContent(name, data) {
			skipped++
			continue
		}

		if dryRun {
			fmt.Printf("  Would import: %s (%d bytes)\n", name, len(data))
			imported++
			continue
		}

		dst := filepath.Join(brainDir, name)
		if err := os.WriteFile(dst, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", dst, err)
			continue
		}
		fmt.Printf("  Imported: %s\n", name)
		imported++
	}

	if dryRun {
		fmt.Printf("\nDry run: %d files would be imported, %d skipped.\n", imported, skipped)
	} else {
		fmt.Printf("\nImported %d topic files from docs/brain/ (%d skipped).\n", imported, skipped)
	}
}

func isStubContent(name string, data []byte) bool {
	if len(bytes.TrimSpace(data)) == 0 {
		return true
	}
	topicName := strings.TrimSuffix(name, ".md")
	content := string(data)
	switch topicName {
	case "gotchas":
		return strings.HasPrefix(content, "# Gotchas\n<!--") && len(content) < 100
	case "patterns":
		return strings.HasPrefix(content, "# Patterns\n<!--") && len(content) < 100
	case "decisions":
		return strings.HasPrefix(content, "# Decisions\n<!--") && len(content) < 100
	case "architecture":
		return strings.HasPrefix(content, "# Architecture\n<!--") && len(content) < 100
	}
	return false
}
