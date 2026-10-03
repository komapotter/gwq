package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/komapotter/gwq/internal/worktree"
)

func TestHandleRenamePost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inShim        bool
		cwdInside     bool
		samePath      bool
		wantStdout    string
		wantStderrHas []string
	}{
		{
			name:          "shim+cwd inside: path on stdout, msg on stderr",
			inShim:        true,
			cwdInside:     true,
			wantStdout:    "/new/path\n",
			wantStderrHas: []string{"Renamed worktree 'old' → 'new'", "Moved /old/path → /new/path"},
		},
		{
			name:          "shim+cwd outside: stdout empty",
			inShim:        true,
			wantStdout:    "",
			wantStderrHas: []string{"Renamed worktree 'old' → 'new'"},
		},
		{
			name:          "shim+same path: no cd even if cwd inside",
			inShim:        true,
			cwdInside:     true,
			samePath:      true,
			wantStdout:    "",
			wantStderrHas: []string{"Renamed branch 'old' → 'new'"},
		},
		{
			name:       "nonshim: success on stdout",
			wantStdout: "Renamed worktree 'old' → 'new'\nMoved /old/path → /new/path\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			r := renameResult{
				OldBranch: "old",
				NewBranch: "new",
				OldPath:   "/old/path",
				NewPath:   "/new/path",
			}
			if tt.samePath {
				r.NewPath = r.OldPath
			}

			handleRenamePost(&stdout, &stderr, tt.inShim, tt.cwdInside, r)

			if got := stdout.String(); got != tt.wantStdout {
				t.Errorf("stdout = %q; want %q", got, tt.wantStdout)
			}
			for _, sub := range tt.wantStderrHas {
				if !strings.Contains(stderr.String(), sub) {
					t.Errorf("stderr = %q; want to contain %q", stderr.String(), sub)
				}
			}
		})
	}
}

func TestPrintRenameDryRun(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printRenameDryRun(&buf, &worktree.RenamePlan{
		OldBranch: "feature/old",
		NewBranch: "feature/new",
		OldPath:   "/old/path",
		NewPath:   "/new/path",
	})

	got := buf.String()
	for _, want := range []string{
		"Would rename:",
		"branch: feature/old → feature/new",
		"path:   /old/path → /new/path",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output %q; want to contain %q", got, want)
		}
	}
}

func TestPromptBranchName(t *testing.T) {
	t.Parallel()

	t.Run("reads trimmed name", func(t *testing.T) {
		var out bytes.Buffer
		name, err := promptBranchName(strings.NewReader("  feature/new  \n"), &out)
		if err != nil {
			t.Fatalf("promptBranchName() error = %v", err)
		}
		if name != "feature/new" {
			t.Errorf("name = %q, want feature/new", name)
		}
		if !strings.Contains(out.String(), "New branch name:") {
			t.Errorf("prompt = %q, want to ask for branch name", out.String())
		}
	})

	t.Run("empty is error", func(t *testing.T) {
		var out bytes.Buffer
		_, err := promptBranchName(strings.NewReader("\n"), &out)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})
}

func TestCwdInsideWorktree(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	child := filepath.Join(root, "sub")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if !cwdInsideWorktree(root, root) {
		t.Error("worktree root should be inside itself")
	}
	if !cwdInsideWorktree(child, root) {
		t.Error("child should be inside worktree")
	}
	if cwdInsideWorktree(t.TempDir(), root) {
		t.Error("unrelated dir should not be inside worktree")
	}
	if cwdInsideWorktree("", root) {
		t.Error("empty cwd should not be inside")
	}
}
