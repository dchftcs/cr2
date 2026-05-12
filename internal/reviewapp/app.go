package reviewapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/dc/cr2/internal/diffengine"
	"github.com/dc/cr2/internal/domain"
)

type Repository interface {
	Diff(ctx context.Context, spec DiffSpec) (string, error)
	Status(ctx context.Context) ([]domain.RepoStatus, error)
	Stage(ctx context.Context, path string) error
	Unstage(ctx context.Context, path string) error
	DefaultBranch(ctx context.Context) string
	CurrentBranch(ctx context.Context) (string, error)
	Capabilities() domain.Capabilities
}

type DiffMode int

const (
	DiffWorkingTree DiffMode = iota
	DiffUnstaged
	DiffRevision
)

type DiffSpec struct {
	Mode    DiffMode
	RevSpec string
}

type App struct {
	repo Repository
}

func New(repo Repository) App {
	return App{repo: repo}
}

func (a App) Load(ctx context.Context, spec DiffSpec) (domain.ReviewSession, error) {
	raw, err := a.repo.Diff(ctx, spec)
	if err != nil {
		return domain.ReviewSession{}, err
	}
	files, err := diffengine.ParseUnified(raw)
	if err != nil {
		return domain.ReviewSession{}, err
	}
	statuses, _ := a.repo.Status(ctx)
	applyStatus(files, statuses)
	return domain.NewReviewSession(diffContext(spec), files), nil
}

func (a App) ToggleStage(ctx context.Context, session *domain.ReviewSession, path string) error {
	if !a.repo.Capabilities().CanStage {
		return fmt.Errorf("staging is not available for this repository target")
	}
	staged := false
	for _, f := range session.Files {
		if f.Path() == path {
			staged = f.Staged
			break
		}
	}
	if staged {
		if err := a.repo.Unstage(ctx, path); err != nil {
			return err
		}
	} else {
		if err := a.repo.Stage(ctx, path); err != nil {
			return err
		}
	}
	statuses, err := a.repo.Status(ctx)
	if err != nil {
		return err
	}
	applyStatus(session.Files, statuses)
	return nil
}

func diffContext(spec DiffSpec) domain.DiffContext {
	switch spec.Mode {
	case DiffUnstaged:
		return domain.DiffContext{
			Left:              "index",
			Right:             "working tree",
			IncludesUnstaged:  true,
			IncludesUntracked: true,
		}
	case DiffRevision:
		if strings.Contains(spec.RevSpec, "...") {
			parts := strings.SplitN(spec.RevSpec, "...", 2)
			ctx := domain.DiffContext{
				Left:  fmt.Sprintf("merge-base(%s,%s)", parts[0], parts[1]),
				Right: parts[1],
			}
			if parts[1] == "HEAD" {
				ctx.IncludesStaged = true
				ctx.IncludesUnstaged = true
				ctx.IncludesUntracked = true
			}
			return ctx
		}
		if strings.Contains(spec.RevSpec, "..") {
			parts := strings.SplitN(spec.RevSpec, "..", 2)
			return domain.DiffContext{Left: parts[0], Right: parts[1]}
		}
		return domain.DiffContext{Left: spec.RevSpec + "^", Right: spec.RevSpec}
	default:
		return domain.DiffContext{
			Left:              "HEAD",
			Right:             "working tree",
			IncludesStaged:    true,
			IncludesUnstaged:  true,
			IncludesUntracked: true,
		}
	}
}

func applyStatus(files []domain.FileChange, statuses []domain.RepoStatus) {
	byPath := make(map[string]domain.RepoStatus, len(statuses))
	for _, st := range statuses {
		byPath[st.Path] = st
	}
	for i := range files {
		if st, ok := byPath[files[i].Path()]; ok {
			files[i].Staged = st.Staged
			files[i].Untracked = st.Untracked
		}
	}
}
