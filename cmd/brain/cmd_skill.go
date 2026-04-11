package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/skill"
)

func cmdSkill(args []string) {
	if len(args) < 2 {
		printSkillUsage()
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine current directory.\n")
		os.Exit(1)
	}

	subcmd := args[1]

	switch subcmd {
	case "list":
		cmdSkillList(cwd)
	case "diff":
		cmdSkillDiff()
	case "update":
		cmdSkillUpdate(cwd)
	case "install":
		globalFlag := false
		for _, arg := range args[2:] {
			if arg == "--global" {
				globalFlag = true
			}
		}
		cmdSkillInstall(cwd, globalFlag)
	default:
		fmt.Fprintf(os.Stderr, "Unknown skill subcommand: %s\n\n", subcmd)
		printSkillUsage()
		os.Exit(1)
	}
}

func cmdSkillList(cwd string) {
	infos := skill.ListInstalled(cwd)

	fmt.Println("Agent skill installation status:")
	fmt.Println()

	projectFound := false
	globalFound := false

	for _, info := range infos {
		label := "Project"
		if info.Global {
			label = "Global"
		}

		if info.Installed {
			if info.Global {
				globalFound = true
			} else {
				projectFound = true
			}
		}

		status := "not installed"
		if info.Installed {
			status = "installed"
			if info.Modified {
				status += " (modified)"
			}
		}

		fmt.Printf("  %-8s %-50s %s\n", label, info.Path, status)
	}

	fmt.Println()
	if !projectFound && !globalFound {
		fmt.Println("No skills installed. Run 'brain skill install' to install.")
	}
}

func cmdSkillDiff() {
	diff, err := skill.ShowDiff()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing diff: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(diff))
}

func cmdSkillUpdate(cwd string) {
	if skill.HasUncommittedChanges(cwd) {
		fmt.Println("Warning: You have uncommitted changes to skill files.")
		fmt.Println("Run 'git diff' to review, or 'git stash' to save your changes.")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Continue anyway? [y/N] ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice != "y" && choice != "Y" {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	if err := skill.UpdateSkills(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating skills: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Skill files updated successfully.")
	fmt.Println("Note: Your changes have been overwritten. Use 'git diff' to see what changed.")
}

func cmdSkillInstall(cwd string, global bool) {
	if global {
		results, err := skill.InstallGlobal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error installing global skills: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Installing skills to global directories...")
		for _, r := range results {
			if r.Skipped {
				fmt.Printf("  ✓ %s (already exists, skipped)\n", r.Path)
			} else if r.Written {
				fmt.Printf("  ✓ %s\n", r.Path)
			} else if r.Error != nil {
				fmt.Printf("  ✗ %s: %v\n", r.Path, r.Error)
			}
		}
	} else {
		results := skill.InstallProject(cwd)

		fmt.Println("Installing skills to project directories...")
		for _, r := range results {
			if r.Skipped {
				fmt.Printf("  ✓ %s (already exists, skipped)\n", r.Path)
			} else if r.Written {
				fmt.Printf("  ✓ %s\n", r.Path)
			} else if r.Error != nil {
				fmt.Printf("  ✗ %s: %v\n", r.Path, r.Error)
			}
		}
	}
}

func printSkillUsage() {
	fmt.Println(`Usage: brain skill <subcommand> [flags]

Skill management commands:

  list              Show installed skill locations and versions
  diff              Compare installed files vs latest templates
  update            Update skill files (overwrites with confirmation)
  install           Install skill files to project directories
  install --global  Install to global directories instead

Examples:
  brain skill list
  brain skill diff
  brain skill update
  brain skill install
  brain skill install --global`)
}
