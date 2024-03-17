package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

func main() {
	// Specify the directory you want to scan
	dirPath := "/Users/hbogert/src"

	// Read the directory
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		os.Exit(1)
	}

	// Iterate over the files in the directory
	for _, file := range files {
		if file.IsDir() {
			repoPath := filepath.Join(dirPath, file.Name())
			// Try to open the directory as a Git repository
			repo, err := git.PlainOpen(repoPath)
			if err != nil {
				continue
			}

			// Get the current branch
			ref, err := repo.Head()
			if err != nil {
				fmt.Printf("Error getting HEAD for %s: %s\n", repoPath, err)
				continue
			}

			// Check if the branch is upstreamed
			isUpstreamed, err := isBranchUpstreamed(repo, ref.Name().Short())
			if err != nil {
				fmt.Printf("Error checking if branch is upstreamed for %s: %s\n", repoPath, err)
				continue
			}

			if !isUpstreamed {
				// Fetch the remote repository
				_ = repo.Fetch(&git.FetchOptions{
					RefSpecs: []config.RefSpec{"refs/heads/*:refs/remotes/origin/*"},
				})
				// if err != nil && err == git.NoErrAlreadyUpToDate {
				// 	return false, err
				// }
			}

			// Check if the branch is upstreamed
			isUpstreamed, err = isBranchUpstreamed(repo, ref.Name().Short())
			if err != nil {
				fmt.Printf("Error checking if branch is upstreamed for %s: %s\n", repoPath, err)
				continue
			}

			if !isUpstreamed {
				fmt.Printf("Branch %s in %s is not upstreamed\n", ref.Name().String(), repoPath)
				continue
			}
		}
	}
}

// isBranchUpstreamed checks if the given branch is upstreamed in the origin repo
func isBranchUpstreamed(repo *git.Repository, branchName string) (bool, error) {
	// Get the reference to the remote branch
	remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", branchName), true)
	if err != nil {
		return false, err
	}

	// Get the local branch reference
	localRef, err := repo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err != nil {
		return false, err
	}

	// Compare the HEAD of the local branch with the remote branch
	rObject, err := repo.CommitObject(remoteRef.Hash())

	if err != nil {
		return false, fmt.Errorf("well here %w", err)
	}

	lObject, err := repo.CommitObject(localRef.Hash())
	if err != nil {
		return false, err
	}

	return lObject.IsAncestor(rObject)
}
