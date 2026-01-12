package autofix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// WorkspaceManager manages isolated workspaces for fix execution
type WorkspaceManager struct {
	logger    *logrus.Logger
	baseDir   string
	repoURL   string
	gitBranch string
}

// Workspace represents an isolated workspace for executing fixes
type Workspace struct {
	Path      string
	GitRepo   *git.Repository
	CleanupFn func() error
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager(logger *logrus.Logger, baseDir string) *WorkspaceManager {
	return &WorkspaceManager{
		logger:  logger,
		baseDir: baseDir,
	}
}

// SetRepositoryURL sets the repository URL for git-based workspaces
func (wm *WorkspaceManager) SetRepositoryURL(repoURL string) {
	wm.repoURL = repoURL
}

// SetGitBranch sets the git branch to checkout
func (wm *WorkspaceManager) SetGitBranch(branch string) {
	wm.gitBranch = branch
}

// CreateWorkspace creates an isolated workspace for fix execution
func (wm *WorkspaceManager) CreateWorkspace(ctx context.Context, execCtx *ExecutionContext) (*Workspace, error) {
	wm.logger.Infof("Creating workspace for event %s (type: %s)", execCtx.EventID, execCtx.FixPlanType)

	// 1. Create temporary directory
	tmpDir, err := os.MkdirTemp(wm.baseDir, "autofix-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	wm.logger.Debugf("Created temporary directory: %s", tmpDir)

	workspace := &Workspace{
		Path: tmpDir,
		CleanupFn: func() error {
			return os.RemoveAll(tmpDir)
		},
	}

	// 2. For code changes, clone the repository
	if wm.requiresGit(execCtx.FixPlanType) {
		if wm.repoURL == "" {
			// If no repo URL is set, use the current directory as workspace
			wm.logger.Warn("No repository URL configured, using temp directory as workspace")
			return workspace, nil
		}

		repo, err := wm.cloneRepository(ctx, tmpDir)
		if err != nil {
			workspace.CleanupFn()
			return nil, fmt.Errorf("git clone failed: %w", err)
		}

		workspace.GitRepo = repo
		wm.logger.Infof("Repository cloned to workspace")
	}

	return workspace, nil
}

// cloneRepository clones the repository to the workspace
func (wm *WorkspaceManager) cloneRepository(ctx context.Context, targetDir string) (*git.Repository, error) {
	wm.logger.Infof("Cloning repository %s to %s", wm.repoURL, targetDir)

	cloneOptions := &git.CloneOptions{
		URL:      wm.repoURL,
		Depth:    1, // Shallow clone
		Progress: nil,
	}

	// If a specific branch is set, clone that branch
	if wm.gitBranch != "" {
		cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(wm.gitBranch)
		cloneOptions.SingleBranch = true
	}

	repo, err := git.PlainCloneContext(ctx, targetDir, false, cloneOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	return repo, nil
}

// Cleanup removes the workspace and all its contents
func (wm *WorkspaceManager) Cleanup(workspace *Workspace) error {
	if workspace == nil || workspace.CleanupFn == nil {
		return nil
	}

	wm.logger.Debugf("Cleaning up workspace: %s", workspace.Path)
	return workspace.CleanupFn()
}

// requiresGit determines if the fix type requires git operations
func (wm *WorkspaceManager) requiresGit(fixType types.AutoFixType) bool {
	gitRequiredTypes := []types.AutoFixType{
		types.FixTypeCodeChange,
		types.FixTypeDependencyUpdate,
	}

	for _, t := range gitRequiredTypes {
		if fixType == t {
			return true
		}
	}
	return false
}

// CreateBranch creates a new git branch in the workspace
func (wm *WorkspaceManager) CreateBranch(workspace *Workspace, branchName string) error {
	if workspace.GitRepo == nil {
		return fmt.Errorf("workspace has no git repository")
	}

	wm.logger.Infof("Creating branch: %s", branchName)

	// Get the HEAD reference
	headRef, err := workspace.GitRepo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Create new branch
	branchRef := plumbing.NewHashReference(
		plumbing.NewBranchReferenceName(branchName),
		headRef.Hash(),
	)

	err = workspace.GitRepo.Storer.SetReference(branchRef)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout the new branch
	worktree, err := workspace.GitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	wm.logger.Infof("Branch %s created and checked out", branchName)
	return nil
}

// CommitChanges commits all changes in the workspace
func (wm *WorkspaceManager) CommitChanges(workspace *Workspace, message string) error {
	if workspace.GitRepo == nil {
		return fmt.Errorf("workspace has no git repository")
	}

	wm.logger.Infof("Committing changes with message: %s", message)

	worktree, err := workspace.GitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	err = worktree.AddGlob(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit
	commit, err := worktree.Commit(message, &git.CommitOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	wm.logger.Infof("Changes committed: %s", commit.String())
	return nil
}

// GetWorkspacePath returns the full path to a file in the workspace
func (wm *WorkspaceManager) GetWorkspacePath(workspace *Workspace, relPath string) string {
	return filepath.Join(workspace.Path, relPath)
}
