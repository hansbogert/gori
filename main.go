package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/encoding/gocode/gocodec"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

// ProjectStatus tracks the status of a Git repository
type ProjectStatus struct {
	path              string
	isDirty           bool
	hasStash          bool
	upstreamed        bool
	isDirtySnoozed    bool
	hasStashSnoozed   bool
	upstreamedSnoozed bool
}

// IgnoreConfig represents the structure of the .goriignore.cue file
type IgnoreConfig struct {
	Repos []struct {
		Path   string `json:"path"`
		Snooze struct {
			DirtyWorkdir  string `json:"dirty_workdir,omitempty"`
			Stashes       string `json:"stashes,omitempty"`
			NotUpstreamed string `json:"not_upstreamed,omitempty"`
		} `json:"snooze,omitempty"`
	} `json:"repos"`
}

var showChanges bool

func Main() int {
	main()
	return 0
}

func main() {
	rootCmd := &cobra.Command{
		Use: "gori",
		Run: run,
	}

	rootCmd.Flags().BoolVarP(&showChanges, "stat", "s", false, "stat the files if the work tree is not clean")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error executing command:", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	fmt.Println("Emoji Legend:")
	fmt.Println("  🚧: Dirty working directory")
	fmt.Println("  🗄️: Stashed changes")
	fmt.Println("  📤: Not upstreamed")
	fmt.Println("") // Add a blank line for spacing

	ignoreConfig, err := loadIgnoreConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Println("Error loading ignore config:", err)
		// We can continue without the ignore file
	}

	files, err := os.ReadDir("./")
	if err != nil {
		fmt.Println("Error reading directory:", err)
		os.Exit(1)
	}

	// Collect all project statuses
	var projects []ProjectStatus

	// Iterate over the files in the directory
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		repoPath := filepath.Join("./", file.Name())

		// Try to open the directory as a Git repository
		repo, err := git.PlainOpen(repoPath)
		if err != nil {
			continue
		}

		wt, err := repo.Worktree()
		if err != nil {
			fmt.Printf("%s: could not get worktree: %s\n", repoPath, err)
			continue
		}

		status, err := wt.Status()
		if err != nil {
			fmt.Printf("%s: could not get repo status: %s\n", repoPath, err)
			continue
		}

		project := ProjectStatus{
			path:     repoPath,
			isDirty:  !status.IsClean(),
			hasStash: checkForStashes(repoPath),
		}

		// Only check upstream status if the repo is clean
		if !project.isDirty {
			project.upstreamed = isUpstreamed(repo, repoPath)
		}

		// Store original status before snoozing
		hasIssuesBeforeSnooze := project.isDirty || project.hasStash || !project.upstreamed

		// If there are no issues at all, just continue.
		if !hasIssuesBeforeSnooze {
			continue
		}

		// Apply snooze logic
		applySnooze(repoPath, &project, ignoreConfig)

		// Check for issues *after* snoozing
		hasIssuesAfterSnooze := project.isDirty || project.hasStash || !project.upstreamed

		// If there are still issues after snoozing, then display and add to list.
		if hasIssuesAfterSnooze {
			displayProjectStatus(project)
			projects = append(projects, project)

			// If showing changes was requested, show them now
			if project.isDirty && showChanges {
				fmt.Printf("%s\n", status)
			}
		}
	}

	// Ask if user wants to visit projects
	if len(projects) > 0 {
		visitProjects(projects)
	}
}

// displayProjectStatus outputs the status of a repository with appropriate emojis
func displayProjectStatus(project ProjectStatus) {
	statusLine := project.path + ": "

	if project.isDirty {
		statusLine += "🚧" // Construction emoji for dirty working tree
	}

	if project.hasStash {
		statusLine += "🗄️" // File cabinet emoji for stashes
	}

	if !project.isDirty && !project.upstreamed {
		statusLine += "📤" // Outbox emoji for not upstreamed
	}

	if statusLine != project.path+": " {
		fmt.Println(statusLine)
	}
}

// visitProjects interactively walks through each project with issues
func visitProjects(projects []ProjectStatus) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\nDo you want to visit each project? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		return
	}

	for i, project := range projects {
		fmt.Printf("\nProject %d/%d: %s\n", i+1, len(projects), project.path)

		for {
			fmt.Printf("\n(s)tatus, (i)gnore, (n)ext, (e)xecute shell, (q)uit: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			parts := strings.Fields(input)
			if len(parts) == 0 {
				continue
			}
			command := parts[0]

			switch command {
			case "s":
				repo, _ := git.PlainOpen(project.path)
				wt, _ := repo.Worktree()
				status, _ := wt.Status()
				fmt.Printf("\n%s\n", status)
			case "i":
				if len(parts) < 2 {
					fmt.Println("Usage: i <duration> [check]")
					continue
				}
				durationStr := parts[1]
				check := "all"
				if len(parts) > 2 {
					check = parts[2]
				}
				snoozeCheck(project, durationStr, check)
			case "n":
				goto nextProject
			case "e":
				shell := os.Getenv("SHELL")
				if shell == "" {
					shell = "/bin/bash" // fallback to bash if SHELL is not set
				}
				cmd := exec.Command(shell)
				cmd.Dir = project.path
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("Error starting subshell: %s\n", err)
				}
			case "q":
				return
			default:
				fmt.Println("Invalid command.")
			}
		}
	nextProject:
	}
}

func parseSnoozeDuration(durationStr string) (time.Duration, error) {
	durationStr = strings.TrimSpace(strings.ToLower(durationStr))

	// First, try parsing with time.ParseDuration for units like h, m, s
	d, err := time.ParseDuration(durationStr)
	if err == nil {
		return d, nil
	}

	// If that fails, try our custom format for d, w, m, y
	re := regexp.MustCompile(`^(\d+)([dwmy])$`)
	matches := re.FindStringSubmatch(durationStr)

	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s. Use formats like 1h, 2d, 3w, 4m, 5y", durationStr)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err // Should not happen with the regex
	}

	var duration time.Duration
	switch matches[2] {
	case "d":
		duration = time.Hour * 24 * time.Duration(value)
	case "w":
		duration = time.Hour * 24 * 7 * time.Duration(value)
	case "m":
		// Approximate a month as 30 days
		duration = time.Hour * 24 * 30 * time.Duration(value)
	case "y":
		// Approximate a year as 365 days
		duration = time.Hour * 24 * 365 * time.Duration(value)
	default:
		// This case should not be reached due to regex
		return 0, fmt.Errorf("unsupported duration unit: %s", matches[2])
	}
	return duration, nil
}

func snoozeCheck(project ProjectStatus, durationStr string, check string) {
	config, err := loadIgnoreConfig()
	if err != nil {
		config = &IgnoreConfig{}
	}

	validChecks := []string{"dirty", "stash", "upstream", "all"}
	isValcheck := slices.Contains(validChecks, check)
	if !isValcheck {
		fmt.Println("Invalid check specified.")
		return
	}

	duration, err := parseSnoozeDuration(durationStr)
	if err != nil {
		fmt.Println("Invalid duration format:", err)
		return
	}

	snoozeUntil := time.Now().Add(duration).Format(time.DateTime)

	found := false
	for i, repo := range config.Repos {
		if repo.Path == project.path {
			if check == "all" {
				config.Repos[i].Snooze.DirtyWorkdir = snoozeUntil
				config.Repos[i].Snooze.Stashes = snoozeUntil
				config.Repos[i].Snooze.NotUpstreamed = snoozeUntil
			} else {
				switch check {
				case "dirty":
					config.Repos[i].Snooze.DirtyWorkdir = snoozeUntil
				case "stash":
					config.Repos[i].Snooze.Stashes = snoozeUntil
				case "upstream":
					config.Repos[i].Snooze.NotUpstreamed = snoozeUntil
				}
			}
			found = true
			break
		}
	}

	if !found {
		newRepo := struct {
			Path   string `json:"path"`
			Snooze struct {
				DirtyWorkdir  string `json:"dirty_workdir,omitempty"`
				Stashes       string `json:"stashes,omitempty"`
				NotUpstreamed string `json:"not_upstreamed,omitempty"`
			} `json:"snooze,omitempty"`
		}{
			Path: project.path,
		}
		if check == "all" {
			newRepo.Snooze.DirtyWorkdir = snoozeUntil
			newRepo.Snooze.Stashes = snoozeUntil
			newRepo.Snooze.NotUpstreamed = snoozeUntil
		} else {
			switch check {
			case "dirty":
				newRepo.Snooze.DirtyWorkdir = snoozeUntil
			case "stash":
				newRepo.Snooze.Stashes = snoozeUntil
			case "upstream":
				newRepo.Snooze.NotUpstreamed = snoozeUntil
			}
		}
		config.Repos = append(config.Repos, newRepo)
	}

	// Now, write the updated config back to the file
	ctx := cuecontext.New()
	codec := gocodec.New(ctx, nil)
	val, err := codec.Decode(config)
	if err != nil {
		fmt.Println("Error decoding config:", err)
		return
	}

	b, err := format.Node(val.Syntax())
	if err != nil {
		fmt.Println("Error formatting CUE:", err)
		return
	}

	err = os.WriteFile(".goriignore.cue", b, 0644)
	if err != nil {
		fmt.Println("Error writing ignore file:", err)
	}
}

func loadIgnoreConfig() (*IgnoreConfig, error) {
	ignoreFile := ".goriignore.cue"
	content, err := os.ReadFile(ignoreFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ignoreFile, err)
	}

	ctx := cuecontext.New()
	val := ctx.CompileBytes(content, cue.Filename(ignoreFile))
	if val.Err() != nil {
		return nil, fmt.Errorf("compiling %s: %w", ignoreFile, val.Err())
	}

	var cfg IgnoreConfig
	if err := val.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", ignoreFile, err)
	}

	return &cfg, nil
}

func applySnooze(repoPath string, project *ProjectStatus, config *IgnoreConfig) {
	if config == nil {
		return
	}

	for _, repo := range config.Repos {
		if repo.Path == repoPath {
			if project.isDirty && repo.Snooze.DirtyWorkdir != "" {
				if isSnoozed(repo.Snooze.DirtyWorkdir) {
					project.isDirty = false
					project.isDirtySnoozed = true
				}
			}
			if project.hasStash && repo.Snooze.Stashes != "" {
				if isSnoozed(repo.Snooze.Stashes) {
					project.hasStash = false
					project.hasStashSnoozed = true
				}
			}
			if !project.upstreamed && repo.Snooze.NotUpstreamed != "" {
				if isSnoozed(repo.Snooze.NotUpstreamed) {
					project.upstreamed = true
					project.upstreamedSnoozed = true
				}
			}
		}
	}
}

func isSnoozed(snoozeTime string) bool {
	t, err := time.Parse(time.DateTime, snoozeTime)
	if err != nil {
		fmt.Printf("Error parsing snooze time: %s\n", err)
		return false
	}
	return time.Now().Before(t)
}

// checkForStashes checks if the repository has any stashed changes
func checkForStashes(repoPath string) bool {
	stashPath := filepath.Join(repoPath, ".git", "refs", "stash")
	_, err := os.Stat(stashPath)
	return err == nil
}

// isUpstreamed determines if a current checkout is up to date with its origin
// counterpart, or is part of a mainish branch
func isUpstreamed(repo *git.Repository, repoPath string) bool {
	// Get the current branch
	ref, err := repo.Head()
	if err != nil {
		fmt.Printf("Error getting HEAD for %s: %s\n", repoPath, err)
		return false
	}

	// TODO, we should fallback to see if the commit itself is upstreamed
	if ref.Name().Short() == "HEAD" {
		fmt.Printf("%s: local checkout does not have branch name\n", repoPath)
		return false
	}

	// Check if the branch is upstreamed
	isUpstreamed, err := isBranchUpstreamed(repo, ref.Name().Short(), ref.Name().Short())
	if err != nil && err != plumbing.ErrReferenceNotFound {
		// +state nobranchupstream
		fmt.Printf("%s: Error checking if branch itself is upstreamed: %s\n", repoPath, err)
	}
	if isUpstreamed {
		return true
	}

	// Check if the branch is upstreamed with main
	mainish, mainishErr := getLikelyUpstreamMainishBranch(repo)

	if mainishErr != nil {
		fmt.Printf("%s: could not determine upstream branch: %s\n", repoPath, mainishErr)
		return false
	}

	isUpstreamed, err = isBranchUpstreamed(repo, ref.Name().Short(), mainish)
	if err != nil && err != plumbing.ErrReferenceNotFound {
		fmt.Printf("Error checking if branch is upstreamed into main for %s: %s\n", repoPath, err)
		return false
	}

	if err == plumbing.ErrReferenceNotFound {
		fmt.Printf("%s: origin does not have %s branch\n", repoPath, mainish)
		return false
	}

	if !isUpstreamed {
		return false
	}

	return true
}

// getLikelyUpstreamMainishBranch gets the likely upstream mainish branch, e.g.,
// main or master
func getLikelyUpstreamMainishBranch(repo *git.Repository) (string, error) {
	var mainish string
	refIter, _ := repo.References()
	refIter.ForEach(func(r *plumbing.Reference) error {
		if r.Name().IsRemote() {
			if r.Name().Short() == "origin/master" {
				mainish = "master"
			}

			if r.Name().Short() == "origin/main" {
				mainish = "main"
			}
		}
		return nil
	})

	if mainish == "" {
		return mainish, fmt.Errorf("neither main nor master branch exists")
	}

	return mainish, nil
}

// isBranchUpstreamed checks if the given branch is upstreamed in the origin repo
func isBranchUpstreamed(repo *git.Repository, localBranchName, remoteBranchName string) (bool, error) {
	// Get the local branch reference
	localRef, err := repo.Reference(plumbing.NewBranchReferenceName(localBranchName), true)
	if err != nil {
		return false, fmt.Errorf("could not get local branch: %w", err)
	}

	lObject, err := repo.CommitObject(localRef.Hash())
	if err != nil {
		return false, err
	}

	// Get the reference to the remote branch
	remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", remoteBranchName), true)

	if err != nil {
		return false, err
	}

	rObject, err := repo.CommitObject(remoteRef.Hash())

	if err != nil {
		return false, fmt.Errorf(`cannot get remoteRef, \"origin/%s\" by hash: %w`, remoteBranchName, err)
	}

	return lObject.IsAncestor(rObject)
}
