package gitrepo

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dc/cr2/internal/domain"
	"github.com/dc/cr2/internal/reviewapp"
)

type Local struct{}

func NewLocal() Local {
	return Local{}
}

func (Local) Capabilities() domain.Capabilities {
	return domain.Capabilities{
		CanStage:           true,
		CanReadWorkingTree: true,
		CanOpenEditor:      true,
		CanPollStatus:      true,
	}
}

func (r Local) Diff(ctx context.Context, spec reviewapp.DiffSpec) (string, error) {
	switch spec.Mode {
	case reviewapp.DiffUnstaged:
		base, err := r.git(ctx, "diff", "-M")
		if err != nil {
			return "", err
		}
		untracked, err := r.untrackedDiff(ctx)
		if err != nil {
			return "", err
		}
		return joinDiffs(base, untracked), nil
	case reviewapp.DiffRevision:
		if ok, err := r.HasCommits(ctx); err == nil && !ok {
			return "", fmt.Errorf("revision diffs are unavailable before the first commit")
		}
		if strings.Contains(spec.RevSpec, "...") {
			parts := strings.SplitN(spec.RevSpec, "...", 2)
			mergeBase, err := r.git(ctx, "merge-base", parts[0], parts[1])
			if err != nil {
				return "", err
			}
			mergeBase = strings.TrimSpace(mergeBase)
			if parts[1] == "HEAD" {
				base, err := r.git(ctx, "diff", "-M", mergeBase)
				if err != nil {
					return "", err
				}
				untracked, err := r.untrackedDiff(ctx)
				if err != nil {
					return "", err
				}
				return joinDiffs(base, untracked), nil
			}
			return r.git(ctx, "diff", "-M", mergeBase, parts[1])
		}
		if strings.Contains(spec.RevSpec, "..") {
			return r.git(ctx, "diff", "-M", spec.RevSpec)
		}
		return r.git(ctx, "show", "-M", "--format=", spec.RevSpec)
	default:
		hasCommits, err := r.HasCommits(ctx)
		if err != nil {
			return "", err
		}
		if !hasCommits {
			return r.unbornWorkingTreeDiff(ctx)
		}
		base, err := r.git(ctx, "diff", "-M", "HEAD")
		if err != nil {
			return "", err
		}
		untracked, err := r.untrackedDiff(ctx)
		if err != nil {
			return "", err
		}
		return joinDiffs(base, untracked), nil
	}
}

func (r Local) HasCommits(ctx context.Context) (bool, error) {
	_, err := r.git(ctx, "rev-parse", "--verify", "HEAD^{commit}")
	if err == nil {
		return true, nil
	}
	if isBadRevisionError(err) {
		return false, nil
	}
	return false, err
}

func (r Local) Status(ctx context.Context) ([]domain.RepoStatus, error) {
	out, err := r.git(ctx, "status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}
	var statuses []domain.RepoStatus
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if len(line) < 3 {
			continue
		}
		path := line[3:]
		if strings.Contains(path, " -> ") {
			parts := strings.SplitN(path, " -> ", 2)
			path = parts[1]
		}
		statuses = append(statuses, domain.RepoStatus{
			Path:      path,
			Staged:    line[0] != ' ' && line[0] != '?',
			Untracked: line[0] == '?' && line[1] == '?',
		})
	}
	return statuses, nil
}

func (r Local) Stage(ctx context.Context, path string) error {
	_, err := r.git(ctx, "add", "--", path)
	return err
}

func (r Local) Unstage(ctx context.Context, path string) error {
	_, err := r.git(ctx, "restore", "--staged", "--", path)
	return err
}

func (r Local) DefaultBranch(ctx context.Context) string {
	hasCommits, err := r.HasCommits(ctx)
	if err == nil && !hasCommits {
		out, branchErr := r.git(ctx, "symbolic-ref", "--short", "HEAD")
		if branchErr == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out)
		}
	}
	if _, err := r.git(ctx, "rev-parse", "--verify", "main"); err == nil {
		return "main"
	}
	if _, err := r.git(ctx, "rev-parse", "--verify", "master"); err == nil {
		return "master"
	}
	return "main"
}

func (r Local) CurrentBranch(ctx context.Context) (string, error) {
	out, err := r.git(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(out), err
}

func (r Local) unbornWorkingTreeDiff(ctx context.Context) (string, error) {
	staged, err := r.git(ctx, "diff", "-M", "--cached", "--root")
	if err != nil {
		return "", err
	}
	unstaged, err := r.git(ctx, "diff", "-M")
	if err != nil {
		return "", err
	}
	untracked, err := r.untrackedDiff(ctx)
	if err != nil {
		return "", err
	}
	return joinDiffs(staged, unstaged, untracked), nil
}

func (r Local) untrackedDiff(ctx context.Context) (string, error) {
	out, err := r.git(ctx, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}
	var chunks []string
	for _, p := range strings.Split(strings.TrimSpace(out), "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		diff, err := r.gitAllowExitOne(ctx, "diff", "--no-index", "--", "/dev/null", p)
		if err != nil {
			return "", err
		}
		chunks = append(chunks, diff)
	}
	return strings.Join(chunks, "\n"), nil
}

func (Local) git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

func (Local) gitAllowExitOne(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return string(out), nil
	}
	return "", err
}

func joinDiffs(parts ...string) string {
	var clean []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			clean = append(clean, strings.TrimRight(p, "\n"))
		}
	}
	if len(clean) == 0 {
		return ""
	}
	return strings.Join(clean, "\n") + "\n"
}

func isBadRevisionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Needed a single revision") ||
		strings.Contains(msg, "unknown revision") ||
		strings.Contains(msg, "ambiguous argument") ||
		strings.Contains(msg, "bad revision") ||
		strings.Contains(msg, "not a valid object name")
}
