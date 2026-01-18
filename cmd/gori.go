package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"

	"github.com/hansbogert/gori"
)

var showChanges bool
var concurrency int

func Main() int {
	main()
	return 0
}

func main() {
	rootCmd := &cobra.Command{
		Use:  "gori [path]",
		RunE: run,
		Args: cobra.MaximumNArgs(1),
	}

	rootCmd.Flags().BoolVarP(&showChanges, "stat", "s", false, "stat the files if the work tree is not clean")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 8, "maximum number of concurrent git operations")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println("Emoji Legend:")
	fmt.Println("  🚧: Dirty working directory")
	fmt.Println("  🗄️: Stashed changes")
	fmt.Println("  📤: Not upstreamed")
	fmt.Println("") // Add a blank line for spacing

	// Determine the path to scan - use positional parameter or default to current directory
	scanPath := "./"
	if len(args) > 0 {
		scanPath = args[0]
	}

	ignoreConfig, err := gori.LoadIgnoreConfig(scanPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// Log but continue without the ignore file
		fmt.Fprintf(os.Stderr, "Warning: loading ignore config: %v\n", err)
	}

	files, err := os.ReadDir(scanPath)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", scanPath, err)
	}

	var repoPaths []string
	for _, file := range files {
		if file.IsDir() {
			repoPaths = append(repoPaths, filepath.Join(scanPath, file.Name()))
		}
	}
	slices.Sort(repoPaths)

	type repoResult struct {
		status gori.ProjectStatus
		err    error
	}

	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	results := make(map[string]repoResult)
	done := make(map[string]bool)

	sem := make(chan struct{}, concurrency)

	// one thread that feeds concurrent workers
	go func() {
		for _, path := range repoPaths {
			sem <- struct{}{}
			go func(repoPath string) {
				project := gori.ProjectStatus{}
				defer func() {
					<-sem
					mu.Lock()
					done[repoPath] = true
					mu.Unlock()
					cond.Broadcast()
				}()
				repo, err := git.PlainOpen(repoPath)
				if err != nil {
					mu.Lock()
					results[repoPath] = repoResult{err: fmt.Errorf("opening repo: %w", err)}
					mu.Unlock()
					return
				}

				// // Store original status before snoozing
				// hasIssuesBeforeSnooze := project.isDirty || project.hasStash || !project.upstreamed

				wt, err := repo.Worktree()
				if err != nil {
					mu.Lock()
					results[repoPath] = repoResult{err: fmt.Errorf("getting worktree: %w", err)}
					mu.Unlock()
					return
				}

				status, err := wt.Status()

				if err != nil {
					mu.Lock()
					results[repoPath] = repoResult{err: fmt.Errorf("getting repo status: %w", err)}
					mu.Unlock()
					return
				}

				// It is a git repo, so process it.
				project = gori.NewProject(
					repoPath,
					!status.IsClean(),
					checkForStashes(repoPath),
					isUpstreamed(repo, repoPath),
				)

				if !project.Clean() {
					// Apply snooze logic
					gori.ApplySnooze(repoPath, &project, ignoreConfig, scanPath)

					if project.IsDirty && showChanges {
						project.StatusString = status.String()
					}
				}

				// Store the successful result
				mu.Lock()
				results[repoPath] = repoResult{status: project}
				mu.Unlock()
			}(path)
		}
	}()

	// handle worker results
	var projectsToVisit []gori.ProjectStatus
	for _, repoPath := range repoPaths {
		mu.Lock()
		for !done[repoPath] {
			cond.Wait()
		}
		result, ok := results[repoPath] // Check if a result was actually added
		mu.Unlock()

		if ok && result.err == nil {
			project := result.status
			if project.IsDirty || project.HasStash || !project.Upstreamed {
				displayProjectStatus(project)
				if project.IsDirty && showChanges {
					fmt.Printf("%s\n", project.StatusString)
				}
				projectsToVisit = append(projectsToVisit, project)
			}
		}
	}

	// Ask if user wants to visit projects
	if len(projectsToVisit) > 0 {
		visitProjects(projectsToVisit, scanPath)
	}
	return nil
}

// displayProjectStatus outputs the status of a repository with appropriate emojis
func displayProjectStatus(project gori.ProjectStatus) {
	// Show just the directory name, not the full path
	displayName := filepath.Base(project.Path)
	statusLine := displayName + ": "

	if project.IsDirty {
		statusLine += "🚧" // Construction emoji for dirty working tree
	}

	if project.HasStash {
		statusLine += "🗄️" // File cabinet emoji for stashes
	}

	if !project.IsDirty && !project.Upstreamed {
		statusLine += "📤" // Outbox emoji for not upstreamed
	}

	if statusLine != project.Path+": " {
		fmt.Println(statusLine)
	}
}

// visitProjects interactively walks through each project with issues
func visitProjects(projects []gori.ProjectStatus, scanPath string) {
	reader := bufio.NewReader(os.Stdin)

	for i, project := range projects {

	project:
		for {
			fmt.Printf("\nProject %d/%d: %s\n", i+1, len(projects), filepath.Base(project.Path))
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
				repo, _ := git.PlainOpen(project.Path)
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
				gori.SnoozeCheck(project, durationStr, check, scanPath)
			case "n":
				break project
			case "e":
				shell := os.Getenv("SHELL")
				if shell == "" {
					shell = "/bin/bash" // fallback to bash if SHELL is not set
				}
				cmd := exec.Command(shell)
				cmd.Dir = project.Path
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
	}
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
		fmt.Fprintf(os.Stderr, "%s: local checkout does not have branch name\n", repoPath)
		return false
	}

	// Check if the branch is upstreamed
	isUpstreamed, err := isBranchUpstreamed(repo, ref.Name().Short(), ref.Name().Short())
	if err != nil && err != plumbing.ErrReferenceNotFound {
		// +state nobranchupstream
		fmt.Fprintf(os.Stderr, "%s: Error checking if branch itself is upstreamed: %v\n", repoPath, err)
	}
	if isUpstreamed {
		return true
	}

	// Check if the branch is upstreamed with main
	mainish, mainishErr := getLikelyUpstreamMainishBranch(repo)

	if mainishErr != nil {
		fmt.Fprintf(os.Stderr, "%s: could not determine upstream branch: %v\n", repoPath, mainishErr)
		return false
	}

	isUpstreamed, err = isBranchUpstreamed(repo, ref.Name().Short(), mainish)
	if err != nil && err != plumbing.ErrReferenceNotFound {
		fmt.Fprintf(os.Stderr, "Error checking if branch is upstreamed into main for %s: %v\n", repoPath, err)
		return false
	}

	if err == plumbing.ErrReferenceNotFound {
		fmt.Fprintf(os.Stderr, "%s: origin does not have %s branch\n", repoPath, mainish)
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
	refIter, err := repo.References()
	if err != nil {
		return "", fmt.Errorf("could not get references: %w", err)
	}
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
