package export

import (
	"strings"
	"testing"

	"github.com/dc/cr2/internal/domain"
)

func TestMarkdownIncludesContextAndComment(t *testing.T) {
	session := domain.NewReviewSession(domain.DiffContext{
		Left:             "HEAD",
		Right:            "working tree",
		IncludesUnstaged: true,
	}, []domain.FileChange{{
		NewPath: "a.go",
		Hunks: []domain.Hunk{{
			NewStart: 1,
			NewCount: 1,
			Lines: []domain.Line{{
				Op:      domain.LineInsert,
				NewNum:  1,
				Content: "package main",
			}},
		}},
	}})
	session.AddComment(domain.CommentAnchor{File: "a.go", StartLine: 1}, "looks suspicious")

	got := Markdown(session)
	for _, want := range []string{
		"## Diff Context",
		"## File: `a.go`",
		"### `a.go:1`",
		"> looks suspicious",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}
