package tui

import (
	"fmt"
	"strings"

	"github.com/dc/cr2/internal/domain"
)

type diffRowKind int

const (
	rowKindHunk diffRowKind = iota
	rowKindUnified
	rowKindPair
	rowKindComment
)

type diffRow struct {
	kind    diffRowKind
	text    string
	left    *domain.Line
	right   *domain.Line
	oldLine int
	newLine int
	lineNum int
	op      domain.LineOp
	comment *domain.Comment
}

func (m model) rows() []diffRow {
	f, ok := m.currentFile()
	if !ok {
		return nil
	}
	comments := commentsByLine(m.session.Comments, f.Path())
	if m.view == viewSideBySide {
		return sideBySideRows(f, comments)
	}
	return unifiedRows(f, comments)
}

func unifiedRows(f domain.FileChange, comments map[int][]domain.Comment) []diffRow {
	var rows []diffRow
	for _, h := range f.Hunks {
		rows = append(rows, diffRow{
			kind: rowKindHunk,
			text: fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", h.OldStart, h.OldCount, h.NewStart, h.NewCount, h.Section),
		})
		for _, line := range h.Lines {
			num := line.NewNum
			if line.Op == domain.LineDelete {
				num = line.OldNum
			}
			rows = append(rows, diffRow{
				kind:    rowKindUnified,
				text:    line.Content,
				oldLine: line.OldNum,
				newLine: line.NewNum,
				lineNum: num,
				op:      line.Op,
			})
			if line.Op != domain.LineDelete {
				rows = appendCommentRows(rows, comments[num], num)
			}
		}
	}
	return rows
}

func sideBySideRows(f domain.FileChange, comments map[int][]domain.Comment) []diffRow {
	var rows []diffRow
	for _, h := range f.Hunks {
		rows = append(rows, diffRow{
			kind: rowKindHunk,
			text: fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", h.OldStart, h.OldCount, h.NewStart, h.NewCount, h.Section),
		})
		for _, p := range pairLines(h.Lines) {
			num := 0
			op := domain.LineContext
			oldLine := 0
			newLine := 0
			if p.right != nil {
				num = p.right.NewNum
				newLine = p.right.NewNum
				op = p.right.Op
			} else if p.left != nil {
				num = p.left.OldNum
				op = p.left.Op
			}
			if p.left != nil {
				oldLine = p.left.OldNum
				if newLine == 0 && p.left.NewNum > 0 {
					newLine = p.left.NewNum
				}
			}
			rows = append(rows, diffRow{
				kind:    rowKindPair,
				left:    p.left,
				right:   p.right,
				oldLine: oldLine,
				newLine: newLine,
				lineNum: num,
				op:      op,
			})
			rows = appendCommentRows(rows, comments[num], num)
		}
	}
	return rows
}

func appendCommentRows(rows []diffRow, cs []domain.Comment, num int) []diffRow {
	for _, c := range cs {
		cc := c
		rows = append(rows, diffRow{
			kind:    rowKindComment,
			text:    c.Text,
			newLine: num,
			lineNum: num,
			comment: &cc,
		})
	}
	return rows
}

type linePair struct {
	left  *domain.Line
	right *domain.Line
}

// pairLines groups a flat hunk line list into before/after pairs. Context lines
// appear on both sides. Runs of deletes followed by inserts are zipped together
// so corresponding before/after rows render on the same screen line.
func pairLines(lines []domain.Line) []linePair {
	var pairs []linePair
	i := 0
	for i < len(lines) {
		if lines[i].Op == domain.LineContext {
			l := lines[i]
			pairs = append(pairs, linePair{left: &l, right: &l})
			i++
			continue
		}
		delStart := i
		for i < len(lines) && lines[i].Op == domain.LineDelete {
			i++
		}
		insStart := i
		for i < len(lines) && lines[i].Op == domain.LineInsert {
			i++
		}
		deletes := lines[delStart:insStart]
		inserts := lines[insStart:i]
		n := len(deletes)
		if len(inserts) > n {
			n = len(inserts)
		}
		for k := 0; k < n; k++ {
			var lp linePair
			if k < len(deletes) {
				d := deletes[k]
				lp.left = &d
			}
			if k < len(inserts) {
				ins := inserts[k]
				lp.right = &ins
			}
			pairs = append(pairs, lp)
		}
		if n == 0 {
			// Defensive: a non-context line we didn't classify; skip it.
			i++
		}
	}
	return pairs
}

func commentsByLine(comments []domain.Comment, file string) map[int][]domain.Comment {
	out := make(map[int][]domain.Comment)
	for _, c := range comments {
		if c.Anchor.File != file {
			continue
		}
		line := c.Anchor.StartLine
		if c.Anchor.EndLine > 0 {
			line = c.Anchor.EndLine
		}
		out[line] = append(out[line], c)
	}
	return out
}

func (r diffRow) isSource() bool {
	return r.kind == rowKindUnified || r.kind == rowKindPair
}

func (r diffRow) searchText() string {
	switch r.kind {
	case rowKindPair:
		var parts []string
		if r.left != nil {
			parts = append(parts, r.left.Content)
		}
		if r.right != nil {
			parts = append(parts, r.right.Content)
		}
		return strings.Join(parts, "\n")
	default:
		return r.text
	}
}
