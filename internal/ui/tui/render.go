package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dc/cr2/internal/domain"
)

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62"))
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))
	addStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	delStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hunkStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	commentStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	commentEntryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("236"))
	commentEntryGutterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Background(lipgloss.Color("236"))
	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	relativeNumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("238")).
				Width(3).
				Align(lipgloss.Right)
)

const (
	minLineNumberWidth = 5
	relativeNumWidth   = 3
)

func renderTextRow(row diffRow, width, numW int, showAbsolute bool, sideBySide bool) string {
	var line string
	switch row.kind {
	case rowKindUnified:
		text := expandTabs(row.text, 4)
		prefix := " "
		if row.op == domain.LineInsert {
			prefix = "+"
		}
		if row.op == domain.LineDelete {
			prefix = "-"
		}
		if showAbsolute {
			line = truncate(fmt.Sprintf("%*s %*s %s %s",
				numW, lineNumber(row.oldLine),
				numW, lineNumber(row.newLine),
				prefix,
				text,
			), width)
		} else {
			line = truncate(fmt.Sprintf("%s %s", prefix, text), width)
		}
	case rowKindComment:
		gutter := strings.Repeat(" ", textIndent(numW, showAbsolute, sideBySide))
		line = truncate(gutter+"> "+firstLine(row.text), width)
		return commentStyle.Render(line)
	case rowKindHunk:
		gutter := strings.Repeat(" ", textIndent(numW, showAbsolute, sideBySide))
		line = truncate(gutter+row.text, width)
		return hunkStyle.Render(line)
	default:
		line = truncate(row.text, width)
	}
	switch row.op {
	case domain.LineInsert:
		return addStyle.Render(line)
	case domain.LineDelete:
		return delStyle.Render(line)
	}
	return line
}

func fileHeaderText(f domain.FileChange) string {
	if f.Renamed && f.OldPath != "" && f.NewPath != "" && f.OldPath != f.NewPath {
		return f.OldPath + " → " + f.NewPath
	}
	return f.Path()
}

func renderRenameSubheader(f domain.FileChange, width, relW int) string {
	sep := statusStyle.Render("│")
	columnBudget := max(2, width-1-(2*relW))
	leftW := max(1, columnBudget/2)
	rightW := max(1, columnBudget-leftW)
	relPad := strings.Repeat(" ", relW)
	left := padOrTruncate("renamed from "+f.OldPath, leftW)
	right := padOrTruncate("renamed to "+f.NewPath, rightW)
	return relPad + statusStyle.Render(left) + sep + relPad + statusStyle.Render(right)
}

func renderPair(row diffRow, width, numW int, showAbsolute bool, relPrefix string) string {
	sep := statusStyle.Render("│")
	relW := visibleWidth(relPrefix)
	columnBudget := max(2, width-1-(2*relW))
	leftW := max(1, columnBudget/2)
	rightW := max(1, columnBudget-leftW)
	left := renderSide(row.left, leftW, true, numW, showAbsolute)
	right := renderSide(row.right, rightW, false, numW, showAbsolute)
	return relPrefix + left + sep + relPrefix + right
}

func renderSide(line *domain.Line, width int, isLeft bool, numW int, showAbsolute bool) string {
	if line == nil {
		if showAbsolute {
			return padOrTruncate(sideNumberGutter("", numW), width)
		}
		return strings.Repeat(" ", width)
	}
	var cell string
	if showAbsolute {
		num := line.OldNum
		if !isLeft {
			num = line.NewNum
		}
		gutter := sideNumberGutter(lineNumber(num), numW)
		text := truncate(expandTabs(line.Content, 4), max(1, width-visibleWidth(gutter)))
		cell = gutter + text
	} else {
		cell = truncate(expandTabs(line.Content, 4), width)
	}
	cell = padOrTruncate(cell, width)
	switch line.Op {
	case domain.LineInsert:
		return addStyle.Render(cell)
	case domain.LineDelete:
		return delStyle.Render(cell)
	}
	return cell
}

func lineNumberWidth(f domain.FileChange) int {
	maxLine := 0
	for _, h := range f.Hunks {
		if h.OldStart+h.OldCount-1 > maxLine {
			maxLine = h.OldStart + h.OldCount - 1
		}
		if h.NewStart+h.NewCount-1 > maxLine {
			maxLine = h.NewStart + h.NewCount - 1
		}
		for _, line := range h.Lines {
			if line.OldNum > maxLine {
				maxLine = line.OldNum
			}
			if line.NewNum > maxLine {
				maxLine = line.NewNum
			}
		}
	}
	if maxLine < 1 {
		return 1
	}
	width := 0
	for maxLine > 0 {
		width++
		maxLine /= 10
	}
	if width < 2 {
		return 2
	}
	return width
}

func absoluteLineNumberWidth(f domain.FileChange) int {
	return max(minLineNumberWidth, lineNumberWidth(f))
}

func textIndent(numW int, showAbsolute bool, sideBySide bool) int {
	if !showAbsolute {
		return 2
	}
	if sideBySide {
		return sideNumberGutterWidth(numW)
	}
	return numW*2 + 4
}

func sideNumberGutterWidth(numW int) int {
	return numW + visibleWidth(" │ ")
}

func sideNumberGutter(num string, numW int) string {
	return fmt.Sprintf("%*s │ ", numW, num)
}

func rowOrdinals(rows []diffRow) []int {
	ordinals := make([]int, len(rows))
	ordinal := 0
	for i, row := range rows {
		if row.isSource() {
			ordinal++
		}
		ordinals[i] = ordinal
	}
	return ordinals
}

func relativeLineNumber(rows []diffRow, ordinals []int, rowIdx, cursorIdx int) string {
	if rowIdx < 0 || rowIdx >= len(rows) || rows[rowIdx].kind == rowKindHunk {
		return strings.Repeat(" ", relativeNumWidth)
	}
	if rowIdx == cursorIdx {
		return fmt.Sprintf("%*d", relativeNumWidth, 0)
	}
	if cursorIdx < 0 || cursorIdx >= len(rows) || rowIdx >= len(ordinals) || cursorIdx >= len(ordinals) {
		return fmt.Sprintf("%*d", relativeNumWidth, abs(rowIdx-cursorIdx))
	}
	dist := abs(ordinals[rowIdx] - ordinals[cursorIdx])
	return fmt.Sprintf("%*d", relativeNumWidth, dist)
}

func lineNumber(n int) string {
	if n < 1 {
		return ""
	}
	return fmt.Sprintf("%d", n)
}

func visibleWidth(s string) int {
	return lipgloss.Width(s)
}
