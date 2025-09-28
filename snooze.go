package gori

import (
	"fmt"
	"os"
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
)

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

func SnoozeCheck(project ProjectStatus, durationStr string, check string, scanPath string) {
	config, err := LoadIgnoreConfig(scanPath)
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
		if repo.Path == project.Path {
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
			Path: getRelativePath(project.Path, scanPath),
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

	ignoreFile := filepath.Join(scanPath, ".goriignore.cue")
	err = os.WriteFile(ignoreFile, b, 0644)
	if err != nil {
		fmt.Println("Error writing ignore file:", err)
	}
}

func LoadIgnoreConfig(scanPath string) (*IgnoreConfig, error) {
	ignoreFile := filepath.Join(scanPath, ".goriignore.cue")
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

func ApplySnooze(repoPath string, project *ProjectStatus, config *IgnoreConfig, scanPath string) {
	if config == nil {
		return
	}

	for _, repo := range config.Repos {
		// The repo.Path is relative to the goriignore file location
		// Convert it to an absolute path for comparison
		ignoreFileDir := scanPath
		if !filepath.IsAbs(scanPath) {
			absScanPath, _ := filepath.Abs(scanPath)
			ignoreFileDir = absScanPath
		}

		// Resolve the repo path relative to the goriignore file directory
		resolvedPath := filepath.Join(ignoreFileDir, repo.Path)
		resolvedPath = filepath.Clean(resolvedPath)

		// Also get absolute path for repoPath for comparison
		absRepoPath, _ := filepath.Abs(repoPath)
		absRepoPath = filepath.Clean(absRepoPath)

		if resolvedPath == absRepoPath {
			if project.IsDirty && repo.Snooze.DirtyWorkdir != "" {
				if isSnoozed(repo.Snooze.DirtyWorkdir) {
					project.IsDirty = false
					project.isDirtySnoozed = true
				}
			}
			if project.HasStash && repo.Snooze.Stashes != "" {
				if isSnoozed(repo.Snooze.Stashes) {
					project.HasStash = false
					project.hasStashSnoozed = true
				}
			}
			if !project.Upstreamed && repo.Snooze.NotUpstreamed != "" {
				if isSnoozed(repo.Snooze.NotUpstreamed) {
					project.Upstreamed = true
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

func getRelativePath(projectPath, scanPath string) string {
	// Get absolute paths for both
	absProjectPath, _ := filepath.Abs(projectPath)
	absScanPath, _ := filepath.Abs(scanPath)

	// Get relative path from scan directory to project
	relPath, err := filepath.Rel(absScanPath, absProjectPath)
	if err != nil {
		// Fallback to original path if we can't compute relative path
		return projectPath
	}

	return relPath
}
