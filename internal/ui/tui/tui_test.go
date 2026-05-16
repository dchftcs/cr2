package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestPageScrollsAreViewportOnly(t *testing.T) {
	// Page and half-page keys move the viewport without moving the cursor —
	// same semantics as mouse wheel. Cursor may leave the viewport.
	for _, key := range []string{"ctrl+f", "pgdown", "ctrl+d"} {
		m := testModel()
		m.view = viewUnified
		m.height = 10
		before := m.cursor
		m = pressKeys(t, m, key)
		if m.cursor != before {
			t.Fatalf("%s moved cursor from %d to %d, want viewport-only scroll", key, before, m.cursor)
		}
		if m.scroll == 0 {
			t.Fatalf("%s should advance scroll, got scroll=%d", key, m.scroll)
		}
	}
}

func TestMouseWheelScrollsDiff(t *testing.T) {
	m := testModel()
	m.view = viewUnified
	m.height = 10 // shrink so the diff actually has room to scroll

	next, _ := m.Update(mouseMsg(50, 5, tea.MouseButtonWheelDown, tea.MouseActionPress))
	got, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if got.scroll != wheelStepDiff {
		t.Fatalf("scroll after wheel down = %d, want %d", got.scroll, wheelStepDiff)
	}
	if got.cursor != 0 {
		t.Fatalf("wheel should not move the cursor, cursor = %d", got.cursor)
	}

	next, _ = got.Update(mouseMsg(50, 5, tea.MouseButtonWheelUp, tea.MouseActionPress))
	got = next.(model)
	if got.scroll != 0 {
		t.Fatalf("scroll after wheel up = %d, want 0", got.scroll)
	}
}

func TestMouseWheelMovesFileSelection(t *testing.T) {
	m := testModel()

	next, _ := m.Update(mouseMsg(10, 5, tea.MouseButtonWheelDown, tea.MouseActionPress))
	got, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if got.selected != 1 {
		t.Fatalf("selected after wheel down on file list = %d, want 1", got.selected)
	}

	next, _ = got.Update(mouseMsg(10, 5, tea.MouseButtonWheelUp, tea.MouseActionPress))
	got = next.(model)
	if got.selected != 0 {
		t.Fatalf("selected after wheel up on file list = %d, want 0", got.selected)
	}
}

func TestMouseClickSelectsFile(t *testing.T) {
	m := testModel()

	// File list: bodyContentTopY=2, no scroll, so y=4 maps to file index 2.
	next, _ := m.Update(mouseMsg(10, 4, tea.MouseButtonLeft, tea.MouseActionPress))
	got, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if got.selected != 2 {
		t.Fatalf("selected after click = %d, want 2", got.selected)
	}
	if got.cursor != 0 || got.scroll != 0 {
		t.Fatalf("file change should reset cursor/scroll, cursor=%d scroll=%d", got.cursor, got.scroll)
	}
}

func TestMouseClickMovesDiffCursor(t *testing.T) {
	m := testModel()
	m.view = viewUnified

	// Right panel: y=2 is the file-header row, y=3 is diff row 0.
	next, _ := m.Update(mouseMsg(50, 4, tea.MouseButtonLeft, tea.MouseActionPress))
	got, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if got.cursor != 1 {
		t.Fatalf("cursor after click = %d, want 1", got.cursor)
	}
}

func TestWorkspaceViewFitsTerminalHeight(t *testing.T) {
	// The workspace view (header + bordered panels + footer) must not
	// exceed m.height — otherwise bubbletea drops the top row, the layout
	// shifts up, and every mouse click is off by 1 row vertically.
	for _, h := range []int{20, 24, 40, 60} {
		m := testModel()
		m.height = h
		got := lipgloss.Height(m.workspaceView())
		if got > h {
			t.Fatalf("workspaceView height = %d for terminal height %d, want <= %d", got, h, h)
		}
	}
}

func TestLayoutRegionsTileWorkspaceWithoutGaps(t *testing.T) {
	// Hit-testing relies on the layout rectangles tiling the workspace: the
	// header sits flush above the panels, panels share an edge, and the
	// footer is the last row. If any region drifts, mouse clicks at the
	// seam between regions are silently dropped.
	for _, h := range []int{20, 24, 40, 60} {
		m := testModel()
		m.view = viewUnified
		m.height = h
		l := m.computeLayout()

		if l.header.y != 0 || l.header.x != 0 {
			t.Fatalf("h=%d header should start at origin, got (%d,%d)", h, l.header.x, l.header.y)
		}
		if l.fileListPanel.y != l.header.y+l.header.h {
			t.Fatalf("h=%d fileListPanel.y=%d, want %d (right below header)", h, l.fileListPanel.y, l.header.y+l.header.h)
		}
		if l.diffPanel.y != l.fileListPanel.y || l.diffPanel.h != l.fileListPanel.h {
			t.Fatalf("h=%d panels not vertically aligned: file=%+v diff=%+v", h, l.fileListPanel, l.diffPanel)
		}
		if l.diffPanel.x != l.fileListPanel.x+l.fileListPanel.w {
			t.Fatalf("h=%d diffPanel.x=%d, want %d (right after fileListPanel)", h, l.diffPanel.x, l.fileListPanel.x+l.fileListPanel.w)
		}
		if got := l.fileListPanel.w + l.diffPanel.w; got != m.width {
			t.Fatalf("h=%d panel widths sum to %d, want %d", h, got, m.width)
		}
		if l.footer.y+l.footer.h != m.height {
			t.Fatalf("h=%d footer ends at %d, want %d", h, l.footer.y+l.footer.h, m.height)
		}
		if l.diffRows.y != l.diffHeader.y+l.diffHeader.h {
			t.Fatalf("h=%d diffRows.y=%d, want %d (right below diffHeader)", h, l.diffRows.y, l.diffHeader.y+l.diffHeader.h)
		}
	}
}

func TestLayoutContains(t *testing.T) {
	m := testModel()
	m.view = viewUnified
	l := m.computeLayout()

	// The first file-list content row should be inside fileListItems but
	// outside diffRows, and vice versa for the first diff row.
	firstFileX := l.fileListItems.x + 1
	firstFileY := l.fileListItems.y
	if !l.fileListItems.contains(firstFileX, firstFileY) {
		t.Fatalf("fileListItems should contain (%d,%d): %+v", firstFileX, firstFileY, l.fileListItems)
	}
	if l.diffRows.contains(firstFileX, firstFileY) {
		t.Fatalf("diffRows should not contain (%d,%d): %+v", firstFileX, firstFileY, l.diffRows)
	}

	firstDiffX := l.diffRows.x + 1
	firstDiffY := l.diffRows.y
	if !l.diffRows.contains(firstDiffX, firstDiffY) {
		t.Fatalf("diffRows should contain (%d,%d): %+v", firstDiffX, firstDiffY, l.diffRows)
	}
	if l.fileListItems.contains(firstDiffX, firstDiffY) {
		t.Fatalf("fileListItems should not contain (%d,%d): %+v", firstDiffX, firstDiffY, l.fileListItems)
	}
}

func TestMouseIgnoredInNonNormalMode(t *testing.T) {
	m := testModel()
	m.view = viewUnified
	m.height = 10
	m.mode = modeHelp

	next, _ := m.Update(mouseMsg(50, 5, tea.MouseButtonWheelDown, tea.MouseActionPress))
	got, ok := next.(model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	if got.scroll != 0 || got.selected != 0 || got.cursor != 0 {
		t.Fatalf("help mode should ignore mouse, scroll=%d selected=%d cursor=%d",
			got.scroll, got.selected, got.cursor)
	}
}

func mouseMsg(x, y int, button tea.MouseButton, action tea.MouseAction) tea.MouseMsg {
	return tea.MouseMsg(tea.MouseEvent{X: x, Y: y, Button: button, Action: action})
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
