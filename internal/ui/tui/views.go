package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.mode == modeHelp {
		return m.helpView()
	}
	if m.mode == modeSearch {
		return m.inputPanelView()
	}
	if m.mode == modeGeneral {
		return lipgloss.JoinVertical(lipgloss.Left, m.workspaceView(), m.generalCommentView())
	}
	return m.workspaceView()
}

func (m model) workspaceView() string {
	header := m.header()
	bodyHeight := max(6, m.height-lipgloss.Height(header)-2)
	if m.mode == modeGeneral {
		bodyHeight = max(6, bodyHeight-lipgloss.Height(m.generalCommentView()))
	}
	leftWidth := clamp(28, 44, max(28, m.width/3))
	rightWidth := max(20, m.width-leftWidth-4)
	left := panelStyle.Width(leftWidth).Height(bodyHeight).Render(m.fileListView(leftWidth, bodyHeight))
	right := panelStyle.Width(rightWidth).Height(bodyHeight).Render(m.diffView(rightWidth, bodyHeight))
	footer := statusStyle.Render(m.footer())
	return lipgloss.JoinVertical(lipgloss.Left, header, lipgloss.JoinHorizontal(lipgloss.Top, left, right), footer)
}

func (m model) header() string {
	return headerStyle.Render("cr") + " " +
		statusStyle.Render(fmt.Sprintf("base: %s  target: %s", m.session.Context.Left, m.session.Context.Right))
}

func (m model) footer() string {
	if m.status != "" {
		return m.status
	}
	if m.mode == modeComment {
		return "typing inline comment  ctrl+s save  esc cancel"
	}
	if m.selecting {
		return "selection active  j/k move endpoint  c comment range  v clear  esc clear"
	}
	return "j/k move  h/l file  / search  42G line  J/K hunk  v select  c comment  s save  q quit  ? help"
}

func (m model) fileListView(width, height int) string {
	if len(m.session.Files) == 0 {
		return "No changes."
	}
	var lines []string
	start := fileListStart(m.selected, len(m.session.Files), height)
	end := min(len(m.session.Files), start+height)
	for i := start; i < end; i++ {
		f := m.session.Files[i]
		flags := "   "
		if m.session.ReadFiles[f.Path()] {
			flags = replaceAt(flags, 0, 'x')
		}
		if f.Staged {
			flags = replaceAt(flags, 1, '^')
		}
		if f.Untracked {
			flags = replaceAt(flags, 2, '?')
		}
		line := fmt.Sprintf("%2d [%s] %s", i+1, flags, f.Path())
		line = truncate(line, max(8, width-2))
		if i == m.selected {
			line = selectedStyle.Width(max(1, width-2)).Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m model) diffView(width, height int) string {
	if len(m.session.Files) == 0 {
		return "No changes."
	}
	f, _ := m.currentFile()
	rows := m.rows()
	if len(rows) == 0 {
		return f.Path() + "\nNo textual diff."
	}
	m.ensureCursorVisible()
	start := clamp(0, max(0, len(rows)-1), m.scroll)
	end := min(len(rows), start+height-2)
	rowW := max(1, width-2)
	header := truncate(f.Path(), rowW)
	if m.view == viewSideBySide {
		header += "  " + statusStyle.Render("(before │ after)")
	}
	numW := absoluteLineNumberWidth(f)
	showRelative := m.showRelativeLineNums()
	showAbsolute := m.showAbsoluteLineNums()
	var ordinals []int
	if showRelative {
		ordinals = rowOrdinals(rows)
	}
	out := []string{headerStyle.Render(header)}
	for i := start; i < end; i++ {
		relPrefix := ""
		if showRelative {
			relPrefix = relativeNumStyle.Render(relativeLineNumber(rows, ordinals, i, m.cursor)) + " "
		}
		var line string
		if rows[i].kind == rowKindPair {
			line = renderPair(rows[i], rowW, numW, showAbsolute, relPrefix)
		} else {
			line = relPrefix + renderTextRow(rows[i], max(1, rowW-visibleWidth(relPrefix)), numW, showAbsolute, m.view == viewSideBySide)
		}
		if m.rowInActiveRange(rows[i]) {
			line = selectedStyle.Width(rowW).Render(stripAnsi(line))
		} else if i == m.cursor {
			line = selectedStyle.Width(rowW).Render(stripAnsi(line))
		}
		out = append(out, line)
		if m.mode == modeComment && m.shouldRenderInlineEditor(rows[i]) {
			out = append(out, m.inlineCommentView(rowW, numW, showAbsolute, m.view == viewSideBySide))
		}
	}
	return strings.Join(out, "\n")
}

func (m model) helpView() string {
	return panelStyle.Width(max(20, m.width-2)).Render(`cr keys

j/k, up/down       move in diff; counts work, e.g. 5j
ctrl+d/ctrl+u      half page down/up
PgUp/PgDn          page diff
g/G, Home/End      top/bottom; count+G jumps to source line
J/K, }/{           next/previous hunk
h/l, ]/[, arrows   previous/next file; counts work
/                  search diff
n/N                next/previous search match
Tab                toggle side-by-side / unified view
ctrl+n             cycle line numbers (both/relative/absolute)
c                  add inline comment at selected source line
v                  start/clear block selection
R                  edit general review comment
ctrl+s             submit comment while editing
m                  mark selected file read/unread
a                  stage/unstage selected file
s                  save review and exit
q                  quit without saving
?                  close help`)
}
