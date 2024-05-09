package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"path/filepath"

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

var showChanges bool

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

		if !status.IsClean() {
			fmt.Printf("%s: 🚧\n", repoPath)
			if showChanges {
				fmt.Printf("%s\n", status)
			}

			continue
		}

		isUpstreamed(repo, repoPath)
	}
}

func isUpstreamed(repo *git.Repository, repoPath string) bool {
	// Get the current branch
	ref, err := repo.Head()
	if err != nil {
		fmt.Printf("Error getting HEAD for %s: %s\n", repoPath, err)
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
		fmt.Printf("%s: origin does not have main\n", repoPath)
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
		return false, err
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
