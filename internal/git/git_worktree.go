package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/komapotter/gwq/pkg/models"
)

// ListWorktrees returns a list of all worktrees in the repository.
func (g *Git) ListWorktrees() ([]models.Worktree, error) {
	output, err := g.run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []models.Worktree
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i := 0; i < len(lines); i++ {
		if after, ok := strings.CutPrefix(lines[i], "worktree "); ok {
			path := after

			var branch, commitHash string
			var locked, detached bool

			for j := i + 1; j < len(lines) && !strings.HasPrefix(lines[j], "worktree "); j++ {
				if after, ok := strings.CutPrefix(lines[j], "branch "); ok {
					branch = after
					// Remove refs/heads/ prefix if present
					branch = strings.TrimPrefix(branch, "refs/heads/")
				} else if after, ok := strings.CutPrefix(lines[j], "HEAD "); ok {
					commitHash = after
				} else if strings.HasPrefix(lines[j], "locked") {
					locked = true
				} else if lines[j] == "detached" {
					detached = true
				}
				i = j
			}

			if detached {
				if branch == "" {
					branch = "HEAD"
				}
			} else if branch == "" {
				branch = g.getCurrentBranch(path)
			}

			info, err := os.Stat(path)
			var createdAt time.Time
			if err == nil {
				createdAt = info.ModTime()
			}

			worktrees = append(worktrees, models.Worktree{
				Path:       path,
				Branch:     branch,
				CommitHash: commitHash,
				Locked:     locked,
				Detached:   detached,
				CreatedAt:  createdAt,
			})
		}
	}

	// git lists the main worktree, or the bare repository, first.
	if len(worktrees) > 0 {
		worktrees[0].IsMain = true
	}

	return worktrees, nil
}

// AddWorktree creates a new worktree.
func (g *Git) AddWorktree(path, branch string, createBranch bool) error {
	args := []string{"worktree", "add"}

	if createBranch {
		args = append(args, "-b", branch, path)
	} else {
		args = append(args, path, branch)
	}

	if _, err := g.run(args...); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	return nil
}

// AddWorktreeFromBase creates a new worktree with a branch from a specific base branch.
func (g *Git) AddWorktreeFromBase(path, branch, baseBranch string) error {
	args := []string{"worktree", "add", "-b", branch, path}

	if baseBranch != "" {
		args = append(args, baseBranch)
	}

	if _, err := g.run(args...); err != nil {
		return fmt.Errorf("failed to add worktree from base branch %s: %w", baseBranch, err)
	}

	return nil
}

// RemoveWorktree removes a worktree.
func (g *Git) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	if _, err := g.run(args...); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// PruneWorktrees removes worktree information for deleted directories.
func (g *Git) PruneWorktrees() error {
	if _, err := g.run("worktree", "prune"); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}
	return nil
}

// MoveWorktree moves a worktree directory to a new path.
func (g *Git) MoveWorktree(oldPath, newPath string) error {
	if _, err := g.run("worktree", "move", oldPath, newPath); err != nil {
		return fmt.Errorf("failed to move worktree from %s to %s: %w", oldPath, newPath, err)
	}
	return nil
}

// HasPopulatedSubmodules reports whether the worktree at path has initialized
// submodules. git worktree move rejects these.
func (g *Git) HasPopulatedSubmodules(path string) (bool, error) {
	if _, err := os.Stat(filepath.Join(path, ".gitmodules")); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check .gitmodules: %w", err)
	}

	oldWorkDir := g.workDir
	g.workDir = path
	defer func() { g.workDir = oldWorkDir }()

	output, err := g.run("submodule", "status", "--recursive")
	if err != nil {
		return false, fmt.Errorf("failed to check submodules: %w", err)
	}

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		// '-' prefix means the submodule is not initialized.
		if !strings.HasPrefix(line, "-") {
			return true, nil
		}
	}

	return false, nil
}
