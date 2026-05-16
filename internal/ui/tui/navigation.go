package tui

import (
	"strings"

	"github.com/dc/cr2/internal/domain"
)

func (m *model) moveFile(delta int) {
	if len(m.session.Files) == 0 {
		return
	}
	m.selected = clamp(0, len(m.session.Files)-1, m.selected+delta)
	m.cursor = 0
	m.scroll = 0
	m.selecting = false
}

func (m *model) moveCursor(delta int) {
	rows := m.rows()
	if len(rows) == 0 {
		return
	}
	m.cursor = clamp(0, len(rows)-1, m.cursor+delta)
	m.ensureCursorVisible()
}

func (m *model) jumpTop() {
	m.cursor = 0
	m.ensureCursorVisible()
}

func (m *model) jumpBottom() {
	m.cursor = max(0, len(m.rows())-1)
	m.ensureCursorVisible()
}

func (m *model) jumpToLine(target int) {
	rows := m.rows()
	if len(rows) == 0 || target < 1 {
		return
	}
	best := -1
	lastSource := -1
	for i, row := range rows {
		if !row.isSource() {
			continue
		}
		lastSource = i
		if row.newLine == target {
			m.cursor = i
			m.ensureCursorVisible()
			return
		}
		if best == -1 && row.lineNum > target {
			best = i
		}
	}
	for i, row := range rows {
		if !row.isSource() {
			continue
		}
		if row.oldLine == target || row.lineNum == target {
			m.cursor = i
			m.ensureCursorVisible()
			return
		}
	}
	if best == -1 {
		best = lastSource
	}
	if best == -1 {
		best = 0
	}
	m.cursor = best
	m.ensureCursorVisible()
}

func (m *model) moveHunk(delta int) {
	rows := m.rows()
	if len(rows) == 0 || delta == 0 {
		return
	}
	step := 1
	if delta < 0 {
		step = -1
		delta = -delta
	}
	pos := m.cursor
	for delta > 0 {
		next := -1
		for i := pos + step; i >= 0 && i < len(rows); i += step {
			if rows[i].kind == rowKindHunk {
				next = i
				break
			}
		}
		if next < 0 {
			break
		}
		pos = next
		delta--
	}
	m.cursor = clamp(0, len(rows)-1, pos)
	m.ensureCursorVisible()
}

func (m *model) repeatSearch(delta int) {
	if strings.TrimSpace(m.search) == "" {
		m.status = "No search query"
		return
	}
	step := 1
	if delta < 0 {
		step = -1
		delta = -delta
	}
	for i := 0; i < delta; i++ {
		if !m.findNextSearchMatch(step) {
			m.status = "No matches for " + m.search
			return
		}
	}
}

func (m *model) findNextSearchMatch(step int) bool {
	rows := m.rows()
	if len(rows) == 0 {
		return false
	}
	query := strings.ToLower(strings.TrimSpace(m.search))
	if query == "" {
		return false
	}
	if step >= 0 {
		step = 1
	} else {
		step = -1
	}
	for offset := 1; offset <= len(rows); offset++ {
		i := mod(m.cursor+step*offset, len(rows))
		if strings.Contains(strings.ToLower(rows[i].searchText()), query) {
			m.cursor = i
			m.ensureCursorVisible()
			m.status = ""
			return true
		}
	}
	return false
}

func (m *model) ensureCursorVisible() {
	height := m.diffHeight()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+height {
		m.scroll = max(0, m.cursor-height+1)
	}
}

func (m *model) ensureInlineEditorVisible() {
	height := m.diffHeight()
	editorHeight := 7
	if height <= editorHeight+1 {
		m.ensureCursorVisible()
		return
	}
	maxCursorOffset := height - editorHeight - 1
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor-m.scroll > maxCursorOffset {
		m.scroll = max(0, m.cursor-maxCursorOffset)
	}
}

func (m model) diffHeight() int {
	return max(1, m.computeLayout().diffRows.h)
}

func (m model) pageStep() int {
	return max(1, m.diffHeight()-1)
}

func (m model) halfPageStep() int {
	return max(1, m.diffHeight()/2)
}

func (m model) currentFile() (domain.FileChange, bool) {
	if m.selected < 0 || m.selected >= len(m.session.Files) {
		return domain.FileChange{}, false
	}
	return m.session.Files[m.selected], true
}

func (m model) currentLine() int {
	rows := m.rows()
	if m.cursor < 0 || m.cursor >= len(rows) {
		return 0
	}
	return rows[m.cursor].lineNum
}

func (m model) currentSourceLine() (int, bool) {
	rows := m.rows()
	if m.cursor < 0 || m.cursor >= len(rows) || !rows[m.cursor].isSource() || rows[m.cursor].lineNum < 1 {
		return 0, false
	}
	return rows[m.cursor].lineNum, true
}

func (m model) currentCommentAnchor(file string) (domain.CommentAnchor, bool) {
	line, ok := m.currentSourceLine()
	if !ok {
		return domain.CommentAnchor{}, false
	}
	anchor := domain.CommentAnchor{File: file, StartLine: line}
	if m.selecting && m.selectAnchor > 0 {
		anchor.StartLine = m.selectAnchor
		anchor.EndLine = line
		if anchor.EndLine < anchor.StartLine {
			anchor.StartLine, anchor.EndLine = anchor.EndLine, anchor.StartLine
		}
		if anchor.EndLine == anchor.StartLine {
			anchor.EndLine = 0
		}
	}
	return anchor, true
}

func (m model) activeRange() (string, int, int, bool) {
	if m.mode == modeComment && m.commentAnchor.File != "" {
		start := m.commentAnchor.StartLine
		end := m.commentAnchor.EndLine
		if end == 0 {
			end = start
		}
		return m.commentAnchor.File, min(start, end), max(start, end), true
	}
	if !m.selecting || m.selectAnchor < 1 {
		return "", 0, 0, false
	}
	f, ok := m.currentFile()
	if !ok {
		return "", 0, 0, false
	}
	line := m.currentLine()
	if line < 1 {
		return "", 0, 0, false
	}
	return f.Path(), min(m.selectAnchor, line), max(m.selectAnchor, line), true
}

func (m model) rowInActiveRange(row diffRow) bool {
	if !row.isSource() || row.lineNum < 1 {
		return false
	}
	f, ok := m.currentFile()
	if !ok {
		return false
	}
	file, start, end, active := m.activeRange()
	return active && file == f.Path() && row.lineNum >= start && row.lineNum <= end
}

func (m model) shouldRenderInlineEditor(row diffRow) bool {
	if !row.isSource() || row.lineNum < 1 || m.commentAnchor.File == "" {
		return false
	}
	f, ok := m.currentFile()
	if !ok || f.Path() != m.commentAnchor.File {
		return false
	}
	line := m.commentAnchor.EndLine
	if line == 0 {
		line = m.commentAnchor.StartLine
	}
	return row.lineNum == line
}

// cursorToLine snaps the cursor to the first row whose lineNum matches.
// Used when toggling view modes so the cursor stays on the same source line.
func (m *model) cursorToLine(line int) {
	if line < 1 {
		m.cursor = 0
		m.scroll = 0
		return
	}
	rows := m.rows()
	for i, r := range rows {
		if r.lineNum == line {
			m.cursor = i
			m.ensureCursorVisible()
			return
		}
	}
	m.cursor = clamp(0, max(0, len(rows)-1), m.cursor)
	m.ensureCursorVisible()
}
