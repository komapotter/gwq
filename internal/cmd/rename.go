package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/d-kuro/gwq/internal/registry"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/spf13/cobra"
)

var (
	renameDryRun bool
	renamePath   string
)

// renameCmd represents the rename command.
var renameCmd = &cobra.Command{
	Use:   "rename [worktree] [new-branch]",
	Short: "Rename a worktree branch and directory",
	Long: `Rename a worktree's git branch and move its directory together.

The new directory is generated from the same naming.template / sanitize_chars
rules as 'gwq add', including any per-repository basedir. A worktree that was
created with a custom path is moved onto the template path unless --path is set.

The target is resolved the same way as 'gwq remove' (branch or path). Multiple
matches are an error.

If no arguments are given, a fuzzy finder selects the worktree and you are
prompted for the new branch name.

The remote branch is not renamed or pushed. Existing tmux sessions keep their
old working directory.`,
	Example: `  # Rename by pattern
  gwq rename feature/old feature/new

  # Interactive: fuzzy-find the worktree, then prompt for the new name
  gwq rename

  # Preview the new branch and path
  gwq rename --dry-run feature/old feature/new

  # Keep a custom destination directory while renaming the branch
  gwq rename --path ~/work/custom feature/old feature/new`,
	Args: cobra.MaximumNArgs(2),
	RunE: runRename,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getRemoveCompletions(cmd, args, toComplete)
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)

	renameCmd.Flags().BoolVarP(&renameDryRun, "dry-run", "d", false, "Show rename targets only")
	renameCmd.Flags().StringVar(&renamePath, "path", "", "Override the destination directory")
}

func runRename(cmd *cobra.Command, args []string) error {
	return ExecuteWithArgs(true, func(ctx *CommandContext, cmd *cobra.Command, args []string) error {
		wt, newBranch, err := resolveRenameArgs(ctx, args)
		if err != nil {
			return err
		}

		plan, err := ctx.WorktreeManager.PlanRename(*wt, newBranch, renamePath)
		if err != nil {
			return err
		}

		if renameDryRun {
			w := cmd.OutOrStdout()
			if isCdShimActive() {
				w = cmd.ErrOrStderr()
			}
			printRenameDryRun(w, plan)
			return nil
		}

		// Capture this before the directory moves; getcwd after a move of
		// the current worktree may already report the new path.
		cwd, _ := os.Getwd()
		inside := cwdInsideWorktree(cwd, plan.OldPath)

		if err := ctx.WorktreeManager.Rename(plan); err != nil {
			return err
		}

		if reg, err := registry.New(); err == nil {
			if err := reg.UpdatePathAndBranch(plan.OldPath, plan.NewPath, plan.NewBranch); err != nil {
				return fmt.Errorf("renamed worktree but failed to update registry: %w", err)
			}
		}

		handleRenamePost(
			os.Stdout, os.Stderr,
			isCdShimActive(),
			inside,
			renameResult{
				OldBranch: plan.OldBranch,
				NewBranch: plan.NewBranch,
				OldPath:   plan.OldPath,
				NewPath:   plan.NewPath,
			},
		)
		return nil
	})(cmd, args)
}

func resolveRenameArgs(ctx *CommandContext, args []string) (*models.Worktree, string, error) {
	worktrees, err := ctx.WorktreeManager.List()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	nonMain := filterNonMainWorktrees(worktrees)
	if len(nonMain) == 0 {
		return nil, "", fmt.Errorf("no renameable worktrees found")
	}

	var wt *models.Worktree
	var newBranch string

	switch {
	case len(args) == 0:
		selected, err := ctx.GetFinder().SelectWorktree(nonMain)
		if err != nil {
			return nil, "", fmt.Errorf("worktree selection cancelled")
		}
		wt = selected
		newBranch, err = promptBranchName(os.Stdin, os.Stderr)
		if err != nil {
			return nil, "", err
		}
	case len(args) == 1:
		selected, err := resolveRenameTarget(ctx, args[0])
		if err != nil {
			return nil, "", err
		}
		wt = selected
		newBranch, err = promptBranchName(os.Stdin, os.Stderr)
		if err != nil {
			return nil, "", err
		}
	default:
		selected, err := resolveRenameTarget(ctx, args[0])
		if err != nil {
			return nil, "", err
		}
		wt = selected
		newBranch = args[1]
	}

	return wt, newBranch, nil
}

func resolveRenameTarget(ctx *CommandContext, pattern string) (*models.Worktree, error) {
	matches, err := ctx.WorktreeManager.GetMatchingWorktrees(pattern)
	if err != nil {
		return nil, err
	}

	var candidates []models.Worktree
	for _, wt := range matches {
		if !wt.IsMain {
			candidates = append(candidates, wt)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no worktree found matching pattern: %s", pattern)
	}
	if len(candidates) > 1 {
		var desc []string
		for _, wt := range candidates {
			desc = append(desc, fmt.Sprintf("%s (%s)", wt.Branch, wt.Path))
		}
		return nil, fmt.Errorf("multiple worktrees match pattern %q:\n  %s", pattern, strings.Join(desc, "\n  "))
	}

	return &candidates[0], nil
}

func promptBranchName(in io.Reader, out io.Writer) (string, error) {
	if _, err := fmt.Fprint(out, "New branch name: "); err != nil {
		return "", fmt.Errorf("failed to prompt for branch name: %w", err)
	}

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read branch name: %w", err)
		}
		return "", fmt.Errorf("new branch name is required")
	}

	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		return "", fmt.Errorf("new branch name is required")
	}
	return name, nil
}

func printRenameDryRun(w io.Writer, plan *worktree.RenamePlan) {
	_, _ = fmt.Fprintf(w, "Would rename:\n")
	_, _ = fmt.Fprintf(w, "  branch: %s → %s\n", plan.OldBranch, plan.NewBranch)
	_, _ = fmt.Fprintf(w, "  path:   %s → %s\n", plan.OldPath, plan.NewPath)
}

type renameResult struct {
	OldBranch string
	NewBranch string
	OldPath   string
	NewPath   string
}

// handleRenamePost routes success messages and the new worktree path after a
// successful rename. Under shell integration, messages go to stderr. If the
// current directory is inside the old worktree, stdout carries only the new
// path so the shell wrapper can cd.
func handleRenamePost(stdout, stderr io.Writer, inShim, cwdInside bool, r renameResult) {
	msgDst := stdout
	if inShim {
		msgDst = stderr
	}

	if r.OldPath == r.NewPath {
		_, _ = fmt.Fprintf(msgDst, "Renamed branch '%s' → '%s'\n", r.OldBranch, r.NewBranch)
	} else {
		_, _ = fmt.Fprintf(msgDst, "Renamed worktree '%s' → '%s'\n", r.OldBranch, r.NewBranch)
		_, _ = fmt.Fprintf(msgDst, "Moved %s → %s\n", r.OldPath, r.NewPath)
	}

	if inShim && cwdInside && r.OldPath != r.NewPath {
		_, _ = fmt.Fprintln(stdout, r.NewPath)
	}
}

func cwdInsideWorktree(cwd, worktreePath string) bool {
	if cwd == "" || worktreePath == "" {
		return false
	}

	child := cwd
	parent := worktreePath
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		child = resolved
	}
	if resolved, err := filepath.EvalSymlinks(worktreePath); err == nil {
		parent = resolved
	}

	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}
