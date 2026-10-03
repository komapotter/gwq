package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d-kuro/gwq/pkg/models"
)

// mockGit is a mock implementation of git operations for testing
type mockGit struct {
	worktrees          []models.Worktree
	repoName           string
	repoPath           string
	repoURL            string
	repoURLError       error
	addError           error
	removeError        error
	listError          error
	pruneError         error
	deleteBranchError  error
	renameBranchError  error
	moveWorktreeError  error
	branchExistsError  error
	hasSubmodulesError error
	existingBranches   []string
	hasSubmodules      bool
	renamedBranches    [][2]string
	movedWorktrees     [][2]string
	recentCommits      []models.CommitInfo
	mainRepoPathError  error
}

func (m *mockGit) ListWorktrees() ([]models.Worktree, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.worktrees, nil
}

func (m *mockGit) AddWorktree(path, branch string, createBranch bool) error {
	if m.addError != nil {
		return m.addError
	}
	m.worktrees = append(m.worktrees, models.Worktree{
		Path:   path,
		Branch: branch,
	})
	return nil
}

func (m *mockGit) RemoveWorktree(path string, force bool) error {
	if m.removeError != nil {
		return m.removeError
	}
	var updated []models.Worktree
	for _, wt := range m.worktrees {
		if wt.Path != path {
			updated = append(updated, wt)
		}
	}
	m.worktrees = updated
	return nil
}

func (m *mockGit) PruneWorktrees() error {
	return m.pruneError
}

func (m *mockGit) GetRepositoryName() (string, error) {
	if m.repoName == "" {
		return "test-repo", nil
	}
	return m.repoName, nil
}

func (m *mockGit) GetRecentCommits(path string, limit int) ([]models.CommitInfo, error) {
	return m.recentCommits, nil
}

func (m *mockGit) GetRepositoryURL() (string, error) {
	if m.repoURLError != nil {
		return "", m.repoURLError
	}
	if m.repoURL != "" {
		return m.repoURL, nil
	}
	return "https://github.com/test-user/test-repo.git", nil
}

func (m *mockGit) DeleteBranch(branch string, force bool) error {
	if m.deleteBranchError != nil {
		return m.deleteBranchError
	}
	return nil
}

func (m *mockGit) GetMainRepositoryPath() (string, error) {
	if m.mainRepoPathError != nil {
		return "", m.mainRepoPathError
	}
	if m.repoPath == "" {
		return "/mock/repo/path", nil
	}
	return m.repoPath, nil
}

func (m *mockGit) AddWorktreeFromBase(path, branch, baseBranch string) error {
	if m.addError != nil {
		return m.addError
	}
	m.worktrees = append(m.worktrees, models.Worktree{
		Path:   path,
		Branch: branch,
	})
	return nil
}

func (m *mockGit) RenameBranch(oldName, newName string) error {
	if m.renameBranchError != nil {
		return m.renameBranchError
	}
	m.renamedBranches = append(m.renamedBranches, [2]string{oldName, newName})
	for i := range m.worktrees {
		if m.worktrees[i].Branch == oldName {
			m.worktrees[i].Branch = newName
		}
	}
	updated := make([]string, 0, len(m.existingBranches))
	for _, b := range m.existingBranches {
		if b == oldName {
			updated = append(updated, newName)
		} else {
			updated = append(updated, b)
		}
	}
	m.existingBranches = updated
	return nil
}

func (m *mockGit) MoveWorktree(oldPath, newPath string) error {
	if m.moveWorktreeError != nil {
		return m.moveWorktreeError
	}
	m.movedWorktrees = append(m.movedWorktrees, [2]string{oldPath, newPath})
	for i := range m.worktrees {
		if m.worktrees[i].Path == oldPath {
			m.worktrees[i].Path = newPath
		}
	}
	return nil
}

func (m *mockGit) BranchExists(branch string) (bool, error) {
	if m.branchExistsError != nil {
		return false, m.branchExistsError
	}
	for _, b := range m.existingBranches {
		if b == branch {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockGit) HasPopulatedSubmodules(path string) (bool, error) {
	if m.hasSubmodulesError != nil {
		return false, m.hasSubmodulesError
	}
	return m.hasSubmodules, nil
}

func TestManagerAdd(t *testing.T) {
	tests := []struct {
		name         string
		branch       string
		customPath   string
		createBranch bool
		config       *models.Config
		wantErr      bool
		errContains  string
	}{
		{
			name:   "WithGeneratedPath",
			branch: "feature/test",
			config: &models.Config{
				Worktree: models.WorktreeConfig{
					BaseDir:   t.TempDir(),
					AutoMkdir: true,
				},
			},
			wantErr: false,
		},
		{
			name:       "WithCustomPath",
			branch:     "feature/test",
			customPath: filepath.Join(t.TempDir(), "custom-worktree"),
			config: &models.Config{
				Worktree: models.WorktreeConfig{
					BaseDir:   t.TempDir(),
					AutoMkdir: true,
				},
			},
			wantErr: false,
		},
		{
			name:         "CreateNewBranch",
			branch:       "feature/new",
			createBranch: true,
			config: &models.Config{
				Worktree: models.WorktreeConfig{
					BaseDir:   t.TempDir(),
					AutoMkdir: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockG := &mockGit{}
			m := New(mockG, tt.config)

			_, err := m.Add(tt.branch, tt.customPath, tt.createBranch)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Add() error = %v, want error containing %s", err, tt.errContains)
			}

			if !tt.wantErr {
				// Verify worktree was added
				if len(mockG.worktrees) != 1 {
					t.Errorf("Expected 1 worktree, got %d", len(mockG.worktrees))
				}
			}
		})
	}
}

func TestManagerRemove(t *testing.T) {
	mockG := &mockGit{
		worktrees: []models.Worktree{
			{Path: "/path/to/worktree1", Branch: "feature1"},
			{Path: "/path/to/worktree2", Branch: "feature2"},
		},
	}

	m := New(mockG, &models.Config{})

	// Remove worktree
	err := m.Remove("/path/to/worktree1", false)
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify worktree was removed
	if len(mockG.worktrees) != 1 {
		t.Errorf("Expected 1 worktree after removal, got %d", len(mockG.worktrees))
	}

	if mockG.worktrees[0].Path != "/path/to/worktree2" {
		t.Errorf("Wrong worktree remained: %s", mockG.worktrees[0].Path)
	}
}

func TestManagerList(t *testing.T) {
	expectedWorktrees := []models.Worktree{
		{Path: "/path/1", Branch: "main", IsMain: true},
		{Path: "/path/2", Branch: "feature"},
	}

	mockG := &mockGit{
		worktrees: expectedWorktrees,
	}

	m := New(mockG, &models.Config{})

	worktrees, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(worktrees) != len(expectedWorktrees) {
		t.Errorf("List() returned %d worktrees, want %d", len(worktrees), len(expectedWorktrees))
	}
}

func TestManagerPrune(t *testing.T) {
	mockG := &mockGit{}
	m := New(mockG, &models.Config{})

	err := m.Prune()
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
}

func TestManagerGetWorktreePath(t *testing.T) {
	mockG := &mockGit{
		worktrees: []models.Worktree{
			{Path: "/path/to/feature-test", Branch: "feature/test"},
			{Path: "/path/to/main", Branch: "main"},
			{Path: "/path/to/bugfix", Branch: "bugfix/issue-123"},
		},
	}

	m := New(mockG, &models.Config{})

	tests := []struct {
		name     string
		pattern  string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "MatchBranch",
			pattern:  "feature",
			wantPath: "/path/to/feature-test",
		},
		{
			name:     "MatchPath",
			pattern:  "bugfix",
			wantPath: "/path/to/bugfix",
		},
		{
			name:    "NoMatch",
			pattern: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := m.GetWorktreePath(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorktreePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && path != tt.wantPath {
				t.Errorf("GetWorktreePath() = %s, want %s", path, tt.wantPath)
			}
		})
	}
}

func TestManagerGetMatchingWorktrees(t *testing.T) {
	mockG := &mockGit{
		worktrees: []models.Worktree{
			{Path: "/path/to/feature-test", Branch: "feature/test"},
			{Path: "/path/to/main", Branch: "main"},
			{Path: "/path/to/bugfix", Branch: "bugfix/issue-123"},
			{Path: "/path/to/feature-auth", Branch: "feature/auth"},
			{Path: "/path/to/feature-api", Branch: "feature/api"},
		},
	}

	m := New(mockG, &models.Config{})

	tests := []struct {
		name         string
		pattern      string
		wantCount    int
		wantBranches []string
	}{
		{
			name:         "MatchMultiple",
			pattern:      "feature",
			wantCount:    3,
			wantBranches: []string{"feature/test", "feature/auth", "feature/api"},
		},
		{
			name:         "MatchSingle",
			pattern:      "main",
			wantCount:    1,
			wantBranches: []string{"main"},
		},
		{
			name:         "MatchPath",
			pattern:      "bugfix",
			wantCount:    1,
			wantBranches: []string{"bugfix/issue-123"},
		},
		{
			name:         "NoMatch",
			pattern:      "nonexistent",
			wantCount:    0,
			wantBranches: []string{},
		},
		{
			name:         "CaseInsensitive",
			pattern:      "FEATURE",
			wantCount:    3,
			wantBranches: []string{"feature/test", "feature/auth", "feature/api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := m.GetMatchingWorktrees(tt.pattern)
			if err != nil {
				t.Errorf("GetMatchingWorktrees() unexpected error = %v", err)
				return
			}

			if len(matches) != tt.wantCount {
				t.Errorf("GetMatchingWorktrees() returned %d matches, want %d", len(matches), tt.wantCount)
			}

			// Check that all expected branches are found
			foundBranches := make(map[string]bool)
			for _, wt := range matches {
				foundBranches[wt.Branch] = true
			}

			for _, expectedBranch := range tt.wantBranches {
				if !foundBranches[expectedBranch] {
					t.Errorf("Expected branch %s not found in matches", expectedBranch)
				}
			}
		})
	}
}

func TestManagerValidateWorktreePath(t *testing.T) {
	tests := []struct {
		name      string
		setupPath func() string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "NonExistentPath",
			setupPath: func() string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr: false,
		},
		{
			name: "EmptyDirectory",
			setupPath: func() string {
				dir := filepath.Join(t.TempDir(), "empty")
				_ = os.MkdirAll(dir, 0755)
				return dir
			},
			wantErr: false,
		},
		{
			name: "NonEmptyDirectory",
			setupPath: func() string {
				dir := filepath.Join(t.TempDir(), "nonempty")
				_ = os.MkdirAll(dir, 0755)
				_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
				return dir
			},
			wantErr: true,
			errMsg:  "directory is not empty",
		},
		{
			name: "ExistingFile",
			setupPath: func() string {
				dir := t.TempDir()
				file := filepath.Join(dir, "file")
				_ = os.WriteFile(file, []byte("content"), 0644)
				return file
			},
			wantErr: true,
			errMsg:  "is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(nil, &models.Config{})
			path := tt.setupPath()

			err := m.ValidateWorktreePath(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorktreePath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateWorktreePath() error = %v, want error containing %s", err, tt.errMsg)
			}
		})
	}
}

func TestGenerateWorktreePath(t *testing.T) {
	tests := []struct {
		name               string
		branch             string
		repoName           string
		wantSuffix         string
		repoPath           string
		repositorySettings []models.RepositorySetting
		mainRepoPathError  error
		wantErr            bool
		wantBaseDir        string // if non-empty, overrides "/base" in expected path
	}{
		{
			name:       "BasicTemplate",
			branch:     "feature/test",
			repoName:   "myrepo",
			wantSuffix: "github.com/test-user/test-repo/feature-test",
		},
		{
			name:       "BranchOnly",
			branch:     "main",
			repoName:   "myrepo",
			wantSuffix: "github.com/test-user/test-repo/main",
		},
		{
			name:       "ComplexSanitization",
			branch:     "feature/test:new",
			repoName:   "myrepo",
			wantSuffix: "github.com/test-user/test-repo/feature-test-new",
		},
		{
			name:     "PerRepoBaseDir",
			branch:   "feature/test",
			repoName: "myrepo",
			repoPath: "/mock/repo/path",
			repositorySettings: []models.RepositorySetting{
				{Repository: "/mock/repo/path", BaseDir: "/per-repo-base"},
			},
			wantSuffix:  "github.com/test-user/test-repo/feature-test",
			wantBaseDir: "/per-repo-base",
		},
		{
			name:     "PerRepoBaseDirEmpty",
			branch:   "feature/test",
			repoName: "myrepo",
			repoPath: "/mock/repo/path",
			repositorySettings: []models.RepositorySetting{
				{Repository: "/mock/repo/path", BaseDir: ""},
			},
			wantSuffix: "github.com/test-user/test-repo/feature-test",
		},
		{
			name:              "GetMainRepoPathError",
			branch:            "feature/test",
			repoName:          "myrepo",
			mainRepoPathError: errors.New("git error"),
			repositorySettings: []models.RepositorySetting{
				{Repository: "/mock/repo/path", BaseDir: "/per-repo-base"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockG := &mockGit{
				repoName:          tt.repoName,
				repoPath:          tt.repoPath,
				mainRepoPathError: tt.mainRepoPathError,
			}

			config := &models.Config{
				Worktree: models.WorktreeConfig{
					BaseDir: "/base",
				},
				RepositorySettings: tt.repositorySettings,
			}

			m := New(mockG, config)

			path, err := m.generateWorktreePath(tt.branch)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("generateWorktreePath() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("generateWorktreePath() error = %v", err)
			}

			baseDir := "/base"
			if tt.wantBaseDir != "" {
				baseDir = tt.wantBaseDir
			}
			expectedPath := filepath.Join(baseDir, tt.wantSuffix)
			if path != expectedPath {
				t.Errorf("generateWorktreePath() = %s, want %s", path, expectedPath)
			}
		})
	}
}

func TestManagerAdd_ConfigurableSetupIntegration(t *testing.T) {
	repoDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}
	worktreeDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	srcFile := filepath.Join(repoDir, "copyme.txt")
	if err := os.WriteFile(srcFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}

	cfg := &models.Config{
		Worktree: models.WorktreeConfig{
			BaseDir:   worktreeDir,
			AutoMkdir: true,
		},
		RepositorySettings: []models.RepositorySetting{
			{
				Repository:    repoDir,
				CopyFiles:     []string{"copyme.txt"},
				SetupCommands: []string{"echo test"},
			},
		},
	}

	mockG := &mockGit{repoPath: repoDir}
	m := New(mockG, cfg)

	_, err = m.Add("feature/test", filepath.Join(worktreeDir, "wt1"), false)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	copied := filepath.Join(worktreeDir, "wt1", "copyme.txt")
	if _, err := os.Stat(copied); err != nil {
		t.Errorf("expected file to be copied: %v", err)
	}
}

func TestManagerAdd_SetupFromWorktreeContext(t *testing.T) {
	repoDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}
	worktreeDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	srcFile := filepath.Join(repoDir, "copyme.txt")
	if err := os.WriteFile(srcFile, []byte("from worktree"), 0644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}

	cfg := &models.Config{
		Worktree: models.WorktreeConfig{
			BaseDir:   worktreeDir,
			AutoMkdir: true,
		},
		RepositorySettings: []models.RepositorySetting{
			{
				Repository: repoDir,
				CopyFiles:  []string{"copyme.txt"},
			},
		},
	}

	// repoPath is repoDir but cwd is different — simulates running from worktree
	mockG := &mockGit{repoPath: repoDir}
	m := New(mockG, cfg)

	_, err = m.Add("feature/wt-test", filepath.Join(worktreeDir, "wt1"), false)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	copied := filepath.Join(worktreeDir, "wt1", "copyme.txt")
	if _, err := os.Stat(copied); err != nil {
		t.Errorf("expected file to be copied from worktree context: %v", err)
	}
}

func renameTestConfig(baseDir string) *models.Config {
	return &models.Config{
		Worktree: models.WorktreeConfig{
			BaseDir:   baseDir,
			AutoMkdir: true,
		},
		Naming: models.NamingConfig{
			Template:      "{{.Host}}/{{.Owner}}/{{.Repository}}/{{.Branch}}",
			SanitizeChars: map[string]string{"/": "-", ":": "-"},
		},
	}
}

func TestPlanRename_PathComputation(t *testing.T) {
	baseDir := t.TempDir()
	oldPath := filepath.Join(t.TempDir(), "custom-old")

	tests := []struct {
		name         string
		wt           models.Worktree
		newBranch    string
		customPath   string
		repoPath     string
		repoSettings []models.RepositorySetting
		wantSuffix   string
		wantPath     string
		wantErr      string
	}{
		{
			name: "TemplateFromNewBranch",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "feature/old",
			},
			newBranch:  "feature/new-ui",
			wantSuffix: "github.com/test-user/test-repo/feature-new-ui",
		},
		{
			name: "SanitizeSlashAndColon",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "feature/old",
			},
			newBranch:  "feature/test:new",
			wantSuffix: "github.com/test-user/test-repo/feature-test-new",
		},
		{
			name: "CustomPathWorktreeMovesOntoTemplate",
			wt: models.Worktree{
				Path:   "/custom/created/path",
				Branch: "feature/old",
			},
			newBranch:  "feature/renamed",
			wantSuffix: "github.com/test-user/test-repo/feature-renamed",
		},
		{
			name: "PathOverride",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "feature/old",
			},
			newBranch:  "feature/new-ui",
			customPath: filepath.Join(baseDir, "override-dir"),
			wantPath:   filepath.Join(baseDir, "override-dir"),
		},
		{
			name: "PerRepoBaseDir",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "feature/old",
			},
			newBranch: "feature/new-ui",
			repoPath:  "/mock/repo/path",
			repoSettings: []models.RepositorySetting{
				{Repository: "/mock/repo/path", BaseDir: "/per-repo-base"},
			},
			wantPath: "/per-repo-base/github.com/test-user/test-repo/feature-new-ui",
		},
		{
			name: "RefuseMain",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "main",
				IsMain: true,
			},
			newBranch: "feature/new",
			wantErr:   "cannot rename the main worktree",
		},
		{
			name: "RefuseDetached",
			wt: models.Worktree{
				Path:     oldPath,
				Branch:   "HEAD",
				Detached: true,
			},
			newBranch: "feature/new",
			wantErr:   "cannot rename a detached HEAD worktree",
		},
		{
			name: "RefuseLocked",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "feature/old",
				Locked: true,
			},
			newBranch: "feature/new",
			wantErr:   "cannot rename a locked worktree",
		},
		{
			name: "RefuseExistingBranch",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "feature/old",
			},
			newBranch: "already-taken",
			wantErr:   "branch already exists",
		},
		{
			name: "RefuseSameBranch",
			wt: models.Worktree{
				Path:   oldPath,
				Branch: "feature/old",
			},
			newBranch: "feature/old",
			wantErr:   "branch already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := renameTestConfig(baseDir)
			cfg.RepositorySettings = tt.repoSettings
			mockG := &mockGit{
				repoPath:         tt.repoPath,
				existingBranches: []string{"already-taken"},
			}
			m := New(mockG, cfg)

			plan, err := m.PlanRename(tt.wt, tt.newBranch, tt.customPath)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("PlanRename() expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("PlanRename() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("PlanRename() error = %v", err)
			}

			want := tt.wantPath
			if want == "" {
				want = filepath.Join(baseDir, tt.wantSuffix)
			}
			if plan.NewPath != want {
				t.Errorf("PlanRename() NewPath = %s, want %s", plan.NewPath, want)
			}
			if plan.OldPath != tt.wt.Path {
				t.Errorf("PlanRename() OldPath = %s, want %s", plan.OldPath, tt.wt.Path)
			}
			if plan.NewBranch != tt.newBranch {
				t.Errorf("PlanRename() NewBranch = %s, want %s", plan.NewBranch, tt.newBranch)
			}
		})
	}
}

func TestPlanRename_RefuseOccupiedDestination(t *testing.T) {
	baseDir := t.TempDir()
	dest := filepath.Join(baseDir, "occupied")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	m := New(&mockGit{}, renameTestConfig(baseDir))
	_, err := m.PlanRename(models.Worktree{
		Path:   filepath.Join(t.TempDir(), "old"),
		Branch: "feature/old",
	}, "feature/new", dest)
	if err == nil {
		t.Fatal("PlanRename() expected occupied destination error")
	}
	if !strings.Contains(err.Error(), "destination path already exists") {
		t.Errorf("PlanRename() error = %v, want occupied destination", err)
	}
}

func TestPlanRename_RefuseSubmodules(t *testing.T) {
	m := New(&mockGit{hasSubmodules: true}, renameTestConfig(t.TempDir()))
	_, err := m.PlanRename(models.Worktree{
		Path:   filepath.Join(t.TempDir(), "old"),
		Branch: "feature/old",
	}, "feature/new", "")
	if err == nil {
		t.Fatal("PlanRename() expected submodule error")
	}
	if !strings.Contains(err.Error(), "submodules") {
		t.Errorf("PlanRename() error = %v, want submodules", err)
	}
}

func TestPlanRename_DryRunDoesNotMutate(t *testing.T) {
	baseDir := t.TempDir()
	oldPath := filepath.Join(t.TempDir(), "old")
	mockG := &mockGit{}
	m := New(mockG, renameTestConfig(baseDir))

	plan, err := m.PlanRename(models.Worktree{
		Path:   oldPath,
		Branch: "feature/old",
	}, "feature/new", "")
	if err != nil {
		t.Fatalf("PlanRename() error = %v", err)
	}
	if plan.NewBranch != "feature/new" {
		t.Errorf("NewBranch = %s, want feature/new", plan.NewBranch)
	}
	if len(mockG.renamedBranches) != 0 {
		t.Errorf("PlanRename() renamed branches = %v, want none", mockG.renamedBranches)
	}
	if len(mockG.movedWorktrees) != 0 {
		t.Errorf("PlanRename() moved worktrees = %v, want none", mockG.movedWorktrees)
	}
}

func TestManagerRename(t *testing.T) {
	baseDir := t.TempDir()
	oldPath := filepath.Join(t.TempDir(), "old")
	mockG := &mockGit{
		worktrees: []models.Worktree{
			{Path: oldPath, Branch: "feature/old"},
		},
		existingBranches: []string{"feature/old"},
	}
	m := New(mockG, renameTestConfig(baseDir))

	plan, err := m.PlanRename(mockG.worktrees[0], "feature/new", "")
	if err != nil {
		t.Fatalf("PlanRename() error = %v", err)
	}
	if err := m.Rename(plan); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	if len(mockG.renamedBranches) != 1 || mockG.renamedBranches[0] != [2]string{"feature/old", "feature/new"} {
		t.Errorf("renamedBranches = %v, want [[feature/old feature/new]]", mockG.renamedBranches)
	}
	if len(mockG.movedWorktrees) != 1 || mockG.movedWorktrees[0][0] != oldPath || mockG.movedWorktrees[0][1] != plan.NewPath {
		t.Errorf("movedWorktrees = %v, want [[%s %s]]", mockG.movedWorktrees, oldPath, plan.NewPath)
	}
	if mockG.worktrees[0].Branch != "feature/new" {
		t.Errorf("worktree branch = %s, want feature/new", mockG.worktrees[0].Branch)
	}
	if mockG.worktrees[0].Path != plan.NewPath {
		t.Errorf("worktree path = %s, want %s", mockG.worktrees[0].Path, plan.NewPath)
	}
}

func TestManagerRename_RollbackWhenMoveFails(t *testing.T) {
	baseDir := t.TempDir()
	oldPath := filepath.Join(t.TempDir(), "old")
	mockG := &mockGit{
		worktrees: []models.Worktree{
			{Path: oldPath, Branch: "feature/old"},
		},
		existingBranches:  []string{"feature/old"},
		moveWorktreeError: errors.New("move failed"),
	}
	m := New(mockG, renameTestConfig(baseDir))

	plan, err := m.PlanRename(mockG.worktrees[0], "feature/new", "")
	if err != nil {
		t.Fatalf("PlanRename() error = %v", err)
	}
	err = m.Rename(plan)
	if err == nil {
		t.Fatal("Rename() expected move error")
	}
	if !strings.Contains(err.Error(), "failed to move worktree") {
		t.Errorf("Rename() error = %v, want move failure", err)
	}

	if len(mockG.renamedBranches) != 2 {
		t.Fatalf("renamedBranches = %v, want rename then rollback", mockG.renamedBranches)
	}
	if mockG.renamedBranches[0] != [2]string{"feature/old", "feature/new"} {
		t.Errorf("first rename = %v, want [feature/old feature/new]", mockG.renamedBranches[0])
	}
	if mockG.renamedBranches[1] != [2]string{"feature/new", "feature/old"} {
		t.Errorf("rollback rename = %v, want [feature/new feature/old]", mockG.renamedBranches[1])
	}
	if mockG.worktrees[0].Branch != "feature/old" {
		t.Errorf("worktree branch after rollback = %s, want feature/old", mockG.worktrees[0].Branch)
	}
	if mockG.worktrees[0].Path != oldPath {
		t.Errorf("worktree path after failed move = %s, want %s", mockG.worktrees[0].Path, oldPath)
	}
}

func TestManagerRename_RollbackFailureIsReported(t *testing.T) {
	baseDir := t.TempDir()
	mockG := &mockGit{
		worktrees: []models.Worktree{
			{Path: filepath.Join(t.TempDir(), "old"), Branch: "feature/old"},
		},
		moveWorktreeError: errors.New("move failed"),
	}
	m := New(mockG, renameTestConfig(baseDir))

	plan, err := m.PlanRename(mockG.worktrees[0], "feature/new", "")
	if err != nil {
		t.Fatalf("PlanRename() error = %v", err)
	}

	calls := 0
	m.git = &renameFailOnSecond{mockGit: mockG, failOn: 2, calls: &calls}

	err = m.Rename(plan)
	if err == nil {
		t.Fatal("Rename() expected combined error")
	}
	if !strings.Contains(err.Error(), "failed to restore branch name") {
		t.Errorf("Rename() error = %v, want rollback failure", err)
	}
}

type renameFailOnSecond struct {
	*mockGit
	failOn int
	calls  *int
}

func (r *renameFailOnSecond) RenameBranch(oldName, newName string) error {
	*r.calls++
	if *r.calls == r.failOn {
		return errors.New("rollback failed")
	}
	return r.mockGit.RenameBranch(oldName, newName)
}

func TestManagerRename_SamePathSkipsMove(t *testing.T) {
	oldPath := filepath.Join(t.TempDir(), "same")
	mockG := &mockGit{
		worktrees: []models.Worktree{
			{Path: oldPath, Branch: "feature/old"},
		},
	}
	m := New(mockG, renameTestConfig(t.TempDir()))

	plan := &RenamePlan{
		OldBranch: "feature/old",
		NewBranch: "feature/new",
		OldPath:   oldPath,
		NewPath:   oldPath,
	}
	if err := m.Rename(plan); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if len(mockG.movedWorktrees) != 0 {
		t.Errorf("movedWorktrees = %v, want none when paths match", mockG.movedWorktrees)
	}
	if len(mockG.renamedBranches) != 1 {
		t.Errorf("renamedBranches = %v, want one branch rename", mockG.renamedBranches)
	}
}
