package git

import (
	"fmt"
	"strings"
	"time"

	"github.com/komapotter/gwq/pkg/models"
)

// ListBranches returns a list of all branches.
func (g *Git) ListBranches(includeRemote bool) ([]models.Branch, error) {
	args := []string{"branch", "-v", "--format=%(refname:short)|%(HEAD)|%(committerdate:iso)|%(objectname)|%(subject)|%(authorname)"}
	if includeRemote {
		args = append(args, "-a")
	}

	output, err := g.run(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []models.Branch
	lines := strings.SplitSeq(strings.TrimSpace(output), "\n")

	for line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		name := parts[0]
		isCurrent := parts[1] == "*"
		dateStr := parts[2]
		hash := parts[3]
		message := parts[4]
		author := parts[5]

		isRemote := strings.HasPrefix(name, "remotes/")
		if isRemote {
			name = strings.TrimPrefix(name, "remotes/")
		}

		date, _ := time.Parse("2006-01-02 15:04:05 -0700", dateStr)

		branches = append(branches, models.Branch{
			Name:      name,
			IsCurrent: isCurrent,
			IsRemote:  isRemote,
			LastCommit: models.CommitInfo{
				Hash:    hash,
				Message: message,
				Author:  author,
				Date:    date,
			},
		})
	}

	return branches, nil
}

// RenameBranch renames a local branch. Git keeps upstream config under the new name.
func (g *Git) RenameBranch(oldName, newName string) error {
	if _, err := g.run("branch", "-m", oldName, newName); err != nil {
		return fmt.Errorf("failed to rename branch %s to %s: %w", oldName, newName, err)
	}
	return nil
}

// BranchExists reports whether a local branch with the given name exists.
func (g *Git) BranchExists(branch string) (bool, error) {
	output, err := g.run("for-each-ref", "--format=%(refname:short)", "--", "refs/heads/"+branch)
	if err != nil {
		return false, fmt.Errorf("failed to check if branch %s exists: %w", branch, err)
	}

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == branch {
			return true, nil
		}
	}

	return false, nil
}

// DeleteBranch deletes a branch.
func (g *Git) DeleteBranch(branch string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)

	if _, err := g.run(args...); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}

	return nil
}

// getCurrentBranch returns the current branch name for a specific worktree.
func (g *Git) getCurrentBranch(worktreePath string) string {
	oldWorkDir := g.workDir
	g.workDir = worktreePath
	defer func() { g.workDir = oldWorkDir }()

	output, err := g.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}
