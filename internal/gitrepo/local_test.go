package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dc/cr2/internal/reviewapp"
)

func TestDiffWorkingTreeBeforeFirstCommitIncludesUntrackedFiles(t *testing.T) {
	repo := initRepo(t)
	writeFile(t, repo, "new.txt", "hello\n")
	chdir(t, repo)

	raw, err := NewLocal().Diff(context.Background(), reviewapp.DiffSpec{Mode: reviewapp.DiffWorkingTree})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "+++ b/new.txt") {
		t.Fatalf("diff does not include untracked file:\n%s", raw)
	}
}

func TestDiffWorkingTreeBeforeFirstCommitIncludesStagedFiles(t *testing.T) {
	repo := initRepo(t)
	writeFile(t, repo, "staged.txt", "hello\n")
	gitRun(t, repo, "add", "staged.txt")
	chdir(t, repo)

	raw, err := NewLocal().Diff(context.Background(), reviewapp.DiffSpec{Mode: reviewapp.DiffWorkingTree})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "+++ b/staged.txt") {
		t.Fatalf("diff does not include staged file:\n%s", raw)
	}
}

func TestHasCommitsFalseBeforeFirstCommit(t *testing.T) {
	repo := initRepo(t)
	chdir(t, repo)

	hasCommits, err := NewLocal().HasCommits(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hasCommits {
		t.Fatal("expected unborn repository to have no commits")
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitRun(t, repo, "init", "--initial-branch", "main")
	return repo
}

func writeFile(t *testing.T, repo, relPath, content string) {
	t.Helper()
	full := filepath.Join(repo, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}
