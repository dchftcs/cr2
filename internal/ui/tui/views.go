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
	l := m.computeLayout()
	leftW := l.fileListItems.w
	rightW := l.diffContent.w
	bodyHeight := l.fileListItems.h
	if m.mode == modeGeneral {
		// Shrink the panel bodies to leave room for the general-comment overlay.
		// This is a render-only adjustment; mouse is disabled in modeGeneral.
		bodyHeight = max(minPanelContentHeight, bodyHeight-lipgloss.Height(m.generalCommentView()))
	}
	left := panelStyle.Width(leftW).Height(bodyHeight).Render(m.fileListView(leftW, bodyHeight))
	right := panelStyle.Width(rightW).Height(bodyHeight).Render(m.diffView(rightW, bodyHeight))
	footer := statusStyle.Render(m.footer())
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), lipgloss.JoinHorizontal(lipgloss.Top, left, right), footer)
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
	rowW := max(1, width-2)
	header := truncate(fileHeaderText(f), rowW)
	if m.view == viewSideBySide {
		header += "  " + statusStyle.Render("(before │ after)")
	}
	showRelative := m.showRelativeLineNums()
	out := []string{headerStyle.Render(header)}
	if f.Renamed && m.view == viewSideBySide {
		relW := 0
		if showRelative {
			relW = relativeNumWidth + 1
		}
		out = append(out, renderRenameSubheader(f, rowW, relW))
	}
	if len(rows) == 0 {
		if f.Renamed {
			out = append(out, statusStyle.Render("Pure rename — no textual diff."))
		} else {
			out = append(out, "No textual diff.")
		}
		return strings.Join(out, "\n")
	}
	start := clamp(0, max(0, len(rows)-1), m.scroll)
	end := min(len(rows), start+height-2)
	numW := absoluteLineNumberWidth(f)
	showAbsolute := m.showAbsoluteLineNums()
	var ordinals []int
	if showRelative {
		ordinals = rowOrdinals(rows)
	}
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
mouse click        focus a file or diff row
mouse wheel        scroll diff / step through file list
s                  save review and exit
q                  quit without saving
?                  close help`)
}
