package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dc/cr2/internal/domain"
	"github.com/dc/cr2/internal/export"
)

// Across multiple files, navigate to source row each time, then comment.
func TestRealisticCrossFile(t *testing.T) {
	m := testModel()

	flow := func(fileIdx int, comment string) {
		// jump to file
		for m.selected != fileIdx {
			next, _ := m.Update(runeKey("l"))
			m = next.(model)
		}
		// move cursor to first source row
		for !m.rows()[m.cursor].isSource() {
			next, _ := m.Update(runeKey("j"))
			m = next.(model)
		}
		// open editor
		next, _ := m.Update(runeKey("c"))
		m = next.(model)
		if m.mode != modeComment {
			t.Fatalf("file %d: not modeComment, status=%q", fileIdx, m.status)
		}
		// type comment
		for _, r := range comment {
			next, _ = m.Update(runeKey(string(r)))
			m = next.(model)
		}
		// save
		next, _ = m.Update(specialKey(tea.KeyCtrlS))
		m = next.(model)
	}

	flow(0, "comment-on-a")
	flow(1, "comment-on-b")
	flow(2, "comment-on-c")

	t.Logf("total comments: %d", len(m.session.Comments))
	for i, c := range m.session.Comments {
		t.Logf("  %d: %s @ %s:%d", i, c.Text, c.Anchor.File, c.Anchor.StartLine)
	}

	md := export.Markdown(m.session)
	for _, want := range []string{"comment-on-a", "comment-on-b", "comment-on-c"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q", want)
		}
	}
	if t.Failed() {
		t.Logf("markdown:\n%s", md)
	}
}

// Edge case: comments on identical-line numbers across files (line 10 in each).
func TestCrossFileSameLineNumber(t *testing.T) {
	m := testModel()
	m.session.AddComment(domain.CommentAnchor{File: "a.go", StartLine: 10}, "in-a")
	m.session.AddComment(domain.CommentAnchor{File: "b.go", StartLine: 10}, "in-b")
	t.Logf("comments: %#v", m.session.Comments)
	md := export.Markdown(m.session)
	if !strings.Contains(md, "in-a") || !strings.Contains(md, "in-b") {
		t.Fatalf("markdown missing comments:\n%s", md)
	}
}
