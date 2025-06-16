package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

// 1. is working dirty >NOK
// 2. if feat branch, is upstream? >OK
// 3. if not upstreamed, is upstreamed in main? if so >OK
// 4. >NOK

type checks struct {
	directory       string
	hasUpstreamFeat *bool
	subsumedInMain  *bool
}

// ProjectStatus tracks the status of a Git repository
type ProjectStatus struct {
	path       string
	isDirty    bool
	hasStash   bool
	upstreamed bool
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

		// Display project status
		displayProjectStatus(project)

		// Only add projects with issues to the list
		if project.isDirty || project.hasStash || !project.upstreamed {
			projects = append(projects, project)
		}

		// If showing changes was requested, show them now
		if project.isDirty && showChanges {
			fmt.Printf("%s\n", status)
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
		statusLine += "⛏️" // Shovel emoji for stashes
	}

	if !project.isDirty && !project.upstreamed {
		statusLine += "❌" // Not upstreamed
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

		if project.isDirty {
			fmt.Printf("Working directory has uncommitted changes. View them? (y/n): ")
			resp, _ := reader.ReadString('\n')
			resp = strings.TrimSpace(strings.ToLower(resp))

			if resp == "y" || resp == "yes" {
				// Show working directory changes using go-git
				repo, _ := git.PlainOpen(project.path)
				wt, _ := repo.Worktree()
				status, _ := wt.Status()
				fmt.Printf("\n%s\n", status)
			}
		}

		if project.hasStash {
			fmt.Printf("Project has stashed changes. View stash list? (y/n): ")
			resp, _ := reader.ReadString('\n')
			resp = strings.TrimSpace(strings.ToLower(resp))

			if resp == "y" || resp == "yes" {
				// Unfortunately, go-git doesn't provide a straightforward way to show stash contents
				// Print the file path so user can navigate to it
				fmt.Printf("\nStash reference found at: %s/.git/refs/stash\n", project.path)
				fmt.Printf("To see stashes, navigate to this directory and run 'git stash list'\n")
			}
		}

		if !project.upstreamed && !project.isDirty {
			fmt.Printf("Branch is not upstreamed. Would you like more details? (y/n): ")
			resp, _ := reader.ReadString('\n')
			resp = strings.TrimSpace(strings.ToLower(resp))

			if resp == "y" || resp == "yes" {
				// This would need a more detailed analysis
				fmt.Printf("\nThe current branch in %s is not upstreamed.\n", project.path)
				fmt.Printf("Use 'git push' to push your changes upstream.\n")
			}
		}

		// Unless it's the last project, ask if the user wants to continue to the next one
		if i < len(projects)-1 {
			fmt.Printf("\nContinue to the next project? (y/n/s for subshell): ")
			cont, _ := reader.ReadString('\n')
			cont = strings.TrimSpace(strings.ToLower(cont))

			if cont == "s" {
				// Start a new subshell in the current project directory
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
				// After subshell exits, ask again about continuing
				fmt.Printf("\nContinue to the next project? (y/n/s for subshell): ")
				cont, _ = reader.ReadString('\n')
				cont = strings.TrimSpace(strings.ToLower(cont))
			}

			if cont != "y" && cont != "yes" {
				break
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
		fmt.Printf("%s: Branch %s is not upstreamed in %s\n", repoPath, ref.Name().String(), mainish)
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
		return false, fmt.Errorf(`cannot get remoteRef, "origin/%s" by hash: %w`, remoteBranchName, err)
	}

	return lObject.IsAncestor(rObject)
}
