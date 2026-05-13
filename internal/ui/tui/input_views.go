package tui

import "strings"

func (m model) inputPanelView() string {
	return panelStyle.Width(max(20, m.width-2)).Render(
		headerStyle.Render("Search") + "\n\n" +
			m.input.View() + "\n\n" +
			statusStyle.Render("enter search  esc cancel"),
	)
}

func (m model) generalCommentView() string {
	input := m.input
	input.SetWidth(max(20, m.width-4))
	return panelStyle.Width(max(20, m.width-2)).Render(
		headerStyle.Render("General Comment") + "\n" +
			input.View() + "\n" +
			statusStyle.Render("ctrl+s save  esc cancel"),
	)
}

func (m model) inlineCommentView(width, numW int, showAbsolute bool, sideBySide bool) string {
	input := m.input
	indent := textIndent(numW, showAbsolute, sideBySide)
	prefixWidth := indent + 2
	inputWidth := max(12, width-prefixWidth)
	input.SetWidth(inputWidth)
	lines := strings.Split(input.View(), "\n")
	for i, line := range lines {
		lines[i] = renderCommentEntryLine(indent, line, inputWidth)
	}
	lines = append(lines, renderCommentEntryLine(indent, "ctrl+s save  esc cancel", inputWidth))
	return strings.Join(lines, "\n")
}

func renderCommentEntryLine(indent int, text string, width int) string {
	prefix := strings.Repeat(" ", indent) + "> "
	return commentEntryGutterStyle.Render(prefix) + commentEntryStyle.Width(width).Render(text)
}
