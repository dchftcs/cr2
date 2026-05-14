package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dc/cr2/internal/domain"
)

func TestNormalModeCountMotionsAndFileNavigation(t *testing.T) {
	m := testModel()
	m.view = viewUnified

	m = pressKeys(t, m, "2", "j")
	if m.cursor != 2 {
		t.Fatalf("cursor after 2j = %d, want 2", m.cursor)
	}

	m = pressKeys(t, m, "2", "l")
	if m.selected != 2 {
		t.Fatalf("selected after 2l = %d, want 2", m.selected)
	}
	if m.cursor != 0 || m.scroll != 0 {
		t.Fatalf("file navigation should reset cursor/scroll, got cursor=%d scroll=%d", m.cursor, m.scroll)
	}

	m = pressKeys(t, m, "h")
	if m.selected != 1 {
		t.Fatalf("selected after h = %d, want 1", m.selected)
	}
}

func TestJumpToLinePrefersNewFileLineNumbers(t *testing.T) {
	m := testModel()
	m.view = viewUnified

	m = pressKeys(t, m, "1", "1", "G")
	row := m.rows()[m.cursor]
	if row.newLine != 11 || row.op != domain.LineInsert {
		t.Fatalf("11G selected new line %d op %v, want inserted new line 11", row.newLine, row.op)
	}

	m = pressKeys(t, m, "4", "2", "G")
	row = m.rows()[m.cursor]
	if row.newLine != 42 {
		t.Fatalf("42G selected new line %d, want 42", row.newLine)
	}
}

func TestHunkNavigation(t *testing.T) {
	m := testModel()
	m.view = viewUnified

	m = pressKeys(t, m, "J")
	if m.cursor != 5 {
		t.Fatalf("next hunk cursor = %d, want 5", m.cursor)
	}

	m = pressKeys(t, m, "K")
	if m.cursor != 0 {
		t.Fatalf("previous hunk cursor = %d, want 0", m.cursor)
	}
}

func TestToggleViewPreservesCurrentLine(t *testing.T) {
	m := testModel()
	m.view = viewUnified
	m.jumpToLine(42)

	m = pressKeys(t, m, "tab")
	if m.view != viewSideBySide {
		t.Fatalf("view = %v, want side-by-side", m.view)
	}
	if got := m.currentLine(); got != 42 {
		t.Fatalf("currentLine after view toggle = %d, want 42", got)
	}
}

func TestSearchNavigationRepeatsForwardAndBackward(t *testing.T) {
	m := testModel()
	m.view = viewUnified
	m.search = "same"

	m = pressKeys(t, m, "n")
	if got := m.currentLine(); got != 10 {
		t.Fatalf("first search match line = %d, want 10", got)
	}

	m = pressKeys(t, m, "n")
	if got := m.currentLine(); got != 12 {
		t.Fatalf("second search match line = %d, want 12", got)
	}

	m = pressKeys(t, m, "N")
	if got := m.currentLine(); got != 10 {
		t.Fatalf("reverse search match line = %d, want 10", got)
	}
}

func TestInlineCommentEditorRendersInsideDiffView(t *testing.T) {
	m := testModel()
	m.view = viewUnified
	m.cursor = 1

	next, _ := m.updateNormal(runeKey("c"))
	m, ok := next.(model)
	if !ok {
		t.Fatalf("updateNormal returned %T", next)
	}
	if m.mode != modeComment {
		t.Fatalf("mode = %v, want comment", m.mode)
	}

	view := stripAnsi(m.View())
	if !strings.Contains(view, "@@ -10,3 +10,3 @@ first") {
		t.Fatalf("comment editor should keep diff visible, view = %q", view)
	}
	if !strings.Contains(view, "ctrl+s save  esc cancel") {
		t.Fatalf("comment editor controls missing from inline view = %q", view)
	}
}

func TestGeneralCommentViewKeepsWorkspaceVisible(t *testing.T) {
	m := testModel()
	m.view = viewUnified

	next, _ := m.updateNormal(runeKey("R"))
	m, ok := next.(model)
	if !ok {
		t.Fatalf("updateNormal returned %T", next)
	}
	if m.mode != modeGeneral {
		t.Fatalf("mode = %v, want general", m.mode)
	}

	view := stripAnsi(m.View())
	for _, want := range []string{"a.go", "@@ -10,3 +10,3 @@ first", "General Comment"} {
		if !strings.Contains(view, want) {
			t.Fatalf("general comment view missing %q in %q", want, view)
		}
	}
}

func TestBlockSelectionCreatesRangeComment(t *testing.T) {
	m := testModel()
	m.view = viewUnified
	m.cursor = 1

	m = pressKeys(t, m, "v", "j")
	next, _ := m.updateNormal(runeKey("c"))
	m, ok := next.(model)
	if !ok {
		t.Fatalf("updateNormal returned %T", next)
	}
	m.input.SetValue("range comment")

	next, _ = m.updateComment(specialKey(tea.KeyCtrlS))
	m, ok = next.(model)
	if !ok {
		t.Fatalf("updateComment returned %T", next)
	}
	if len(m.session.Comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(m.session.Comments))
	}
	got := m.session.Comments[0].Anchor
	if got.File != "a.go" || got.StartLine != 10 || got.EndLine != 11 {
		t.Fatalf("anchor = %#v, want a.go:10-11", got)
	}
	if m.selecting {
		t.Fatal("selection should clear after saving range comment")
	}
}

func TestUnifiedLineNumberGutter(t *testing.T) {
	added := stripAnsi(renderTextRow(diffRow{
		kind:    rowKindUnified,
		text:    "added",
		newLine: 12,
		lineNum: 12,
		op:      domain.LineInsert,
	}, 80, 5, true, false))
	if !strings.Contains(added, "      12 + added") {
		t.Fatalf("insert gutter = %q", added)
	}

	deleted := stripAnsi(renderTextRow(diffRow{
		kind:    rowKindUnified,
		text:    "deleted",
		oldLine: 11,
		lineNum: 11,
		op:      domain.LineDelete,
	}, 80, 5, true, false))
	if !strings.Contains(deleted, "   11       - deleted") {
		t.Fatalf("delete gutter = %q", deleted)
	}
}

func TestSideBySideLineNumberGutterIsSeparateFromIndent(t *testing.T) {
	line := domain.Line{Op: domain.LineContext, OldNum: 272, Content: "\t\tapp := m.app"}
	got := stripAnsi(renderSide(&line, 40, true, 5, true))

	if !strings.HasPrefix(got, "  272 │         app") {
		t.Fatalf("side gutter = %q, want fixed gutter before expanded indentation", got)
	}

	blank := stripAnsi(renderSide(nil, 40, true, 5, true))
	if !strings.HasPrefix(blank, "      │ ") {
		t.Fatalf("blank side gutter = %q", blank)
	}

	pair := stripAnsi(renderPair(diffRow{
		kind:  rowKindPair,
		left:  &domain.Line{Op: domain.LineContext, OldNum: 67, Content: "\tctx      context.Context"},
		right: &domain.Line{Op: domain.LineContext, NewNum: 83, Content: "\tctx      context.Context"},
	}, 90, 5, true, ""))
	if !strings.Contains(pair, "   67 │     ctx") || !strings.Contains(pair, "   83 │     ctx") {
		t.Fatalf("pair gutters = %q", pair)
	}

	hunk := stripAnsi(renderTextRow(diffRow{
		kind: rowKindHunk,
		text: "@@ -1,1 +1,1 @@",
	}, 80, 5, true, true))
	if !strings.HasPrefix(hunk, strings.Repeat(" ", sideNumberGutterWidth(5))+"@@") {
		t.Fatalf("side hunk indent = %q", hunk)
	}
}

func TestRelativeLineNumbersUseDiffRows(t *testing.T) {
	rows := []diffRow{
		{kind: rowKindHunk, text: "@@ -1,2 +1,2 @@"},
		{kind: rowKindUnified, lineNum: 10},
		{kind: rowKindComment, lineNum: 10},
		{kind: rowKindUnified, lineNum: 11},
	}
	ordinals := rowOrdinals(rows)

	if got := relativeLineNumber(rows, ordinals, 0, 1); got != "   " {
		t.Fatalf("hunk relative = %q, want blank", got)
	}
	if got := relativeLineNumber(rows, ordinals, 1, 1); got != "  0" {
		t.Fatalf("cursor relative = %q, want zero", got)
	}
	if got := relativeLineNumber(rows, ordinals, 2, 1); got != "  0" {
		t.Fatalf("comment relative = %q, want same ordinal as source row", got)
	}
	if got := relativeLineNumber(rows, ordinals, 3, 1); got != "  1" {
		t.Fatalf("next source relative = %q, want one", got)
	}
}

func TestSideBySideShowsRelativeNumbersOnBothSides(t *testing.T) {
	m := testModel()
	m.cursor = 1
	m.scroll = 1

	lines := strings.Split(stripAnsi(m.diffView(100, 8)), "\n")
	if len(lines) < 2 {
		t.Fatalf("diff view lines = %d, want at least 2", len(lines))
	}
	if got := strings.Count(lines[1], "  0"); got < 2 {
		t.Fatalf("relative cursor number count = %d in %q, want both sides", got, lines[1])
	}
}

func TestPairLinesHandlesUnevenChanges(t *testing.T) {
	pairs := pairLines([]domain.Line{
		{Op: domain.LineDelete, OldNum: 1, Content: "old one"},
		{Op: domain.LineDelete, OldNum: 2, Content: "old two"},
		{Op: domain.LineInsert, NewNum: 1, Content: "new one"},
		{Op: domain.LineContext, OldNum: 3, NewNum: 2, Content: "same"},
		{Op: domain.LineInsert, NewNum: 3, Content: "extra"},
	})

	if len(pairs) != 4 {
		t.Fatalf("pairs = %d, want 4", len(pairs))
	}
	if pairs[0].left == nil || pairs[0].right == nil {
		t.Fatalf("first pair should zip delete and insert: %#v", pairs[0])
	}
	if pairs[1].left == nil || pairs[1].right != nil {
		t.Fatalf("second pair should be delete-only: %#v", pairs[1])
	}
	if pairs[2].left == nil || pairs[2].right == nil {
		t.Fatalf("third pair should be context on both sides: %#v", pairs[2])
	}
	if pairs[3].left != nil || pairs[3].right == nil {
		t.Fatalf("fourth pair should be insert-only: %#v", pairs[3])
	}
}

func TestPureRenameShowsRenameHeaderWithoutHunks(t *testing.T) {
	pure := domain.FileChange{OldPath: "old.go", NewPath: "new.go", Renamed: true}
	m := model{
		width:  100,
		height: 40,
		session: domain.NewReviewSession(domain.DiffContext{Left: "main", Right: "worktree"}, []domain.FileChange{pure}),
	}
	m.view = viewSideBySide

	view := stripAnsi(m.diffView(100, 12))
	if !strings.Contains(view, "old.go → new.go") {
		t.Fatalf("pure rename header missing arrow: %q", view)
	}
	if !strings.Contains(view, "renamed from old.go") || !strings.Contains(view, "renamed to new.go") {
		t.Fatalf("pure rename missing side labels: %q", view)
	}
	if !strings.Contains(view, "Pure rename") {
		t.Fatalf("pure rename should explain absence of hunks: %q", view)
	}

	m.view = viewUnified
	unifiedView := stripAnsi(m.diffView(100, 12))
	if !strings.Contains(unifiedView, "old.go → new.go") {
		t.Fatalf("unified pure rename header missing arrow: %q", unifiedView)
	}
	if !strings.Contains(unifiedView, "Pure rename") {
		t.Fatalf("unified pure rename missing explanation: %q", unifiedView)
	}
}

func TestRenamedFileShowsRenameAnnotations(t *testing.T) {
	renamed := testFile("new.go")
	renamed.OldPath = "old.go"
	renamed.Renamed = true

	m := model{
		width:  100,
		height: 40,
		session: domain.NewReviewSession(domain.DiffContext{Left: "main", Right: "worktree"}, []domain.FileChange{renamed}),
	}
	m.view = viewSideBySide

	view := stripAnsi(m.diffView(100, 12))
	lines := strings.Split(view, "\n")
	if len(lines) < 2 {
		t.Fatalf("diff view lines = %d, want at least 2", len(lines))
	}
	if !strings.Contains(lines[0], "old.go → new.go") {
		t.Fatalf("header missing rename arrow: %q", lines[0])
	}
	if !strings.Contains(lines[1], "renamed from old.go") || !strings.Contains(lines[1], "renamed to new.go") {
		t.Fatalf("subheader missing rename labels: %q", lines[1])
	}

	m.view = viewUnified
	unifiedLines := strings.Split(stripAnsi(m.diffView(100, 12)), "\n")
	if !strings.Contains(unifiedLines[0], "old.go → new.go") {
		t.Fatalf("unified header missing rename arrow: %q", unifiedLines[0])
	}
	for _, line := range unifiedLines {
		if strings.Contains(line, "renamed from") {
			t.Fatalf("unified view should not show side subheader, got %q", line)
		}
	}
}

func pressKeys(t *testing.T, m model, keys ...string) model {
	t.Helper()
	for _, key := range keys {
		next, _ := m.updateNormal(runeKey(key))
		var ok bool
		m, ok = next.(model)
		if !ok {
			t.Fatalf("updateNormal returned %T", next)
		}
	}
	return m
}

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(s)})
}

func specialKey(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: t})
}

func testModel() model {
	return model{
		width:  100,
		height: 40,
		session: domain.NewReviewSession(domain.DiffContext{
			Left:  "main",
			Right: "worktree",
		}, []domain.FileChange{
			testFile("a.go"),
			testFile("b.go"),
			testFile("c.go"),
		}),
	}
}

func testFile(path string) domain.FileChange {
	return domain.FileChange{
		NewPath: path,
		Hunks: []domain.Hunk{
			{
				OldStart: 10,
				OldCount: 3,
				NewStart: 10,
				NewCount: 3,
				Section:  "first",
				Lines: []domain.Line{
					{Op: domain.LineContext, OldNum: 10, NewNum: 10, Content: "same"},
					{Op: domain.LineDelete, OldNum: 11, Content: "old"},
					{Op: domain.LineInsert, NewNum: 11, Content: "new"},
					{Op: domain.LineContext, OldNum: 12, NewNum: 12, Content: "same again"},
				},
			},
			{
				OldStart: 40,
				OldCount: 3,
				NewStart: 40,
				NewCount: 3,
				Section:  "second",
				Lines: []domain.Line{
					{Op: domain.LineContext, OldNum: 40, NewNum: 40, Content: "near"},
					{Op: domain.LineDelete, OldNum: 41, Content: "before"},
					{Op: domain.LineInsert, NewNum: 42, Content: "after"},
				},
			},
		},
	}
}
