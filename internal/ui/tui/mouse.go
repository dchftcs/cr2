package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

const (
	wheelStepDiff     = 3
	wheelStepFileList = 1
)

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeNormal {
		return m, nil
	}
	l := m.computeLayout()
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.handleWheel(l, msg.X, msg.Y, -1)
	case tea.MouseButtonWheelDown:
		m.handleWheel(l, msg.X, msg.Y, 1)
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			m.handleClick(l, msg.X, msg.Y)
		}
	}
	return m, nil
}

func (m *model) handleWheel(l viewLayout, x, y, dir int) {
	switch {
	case l.fileListPanel.contains(x, y):
		m.moveFile(dir * wheelStepFileList)
	case l.diffPanel.contains(x, y):
		m.scrollDiff(dir * wheelStepDiff)
	}
}

func (m *model) handleClick(l viewLayout, x, y int) {
	switch {
	case l.fileListItems.contains(x, y):
		m.clickFileList(l, y-l.fileListItems.y)
	case l.diffRows.contains(x, y):
		m.clickDiff(y - l.diffRows.y)
	}
}

func (m *model) clickFileList(l viewLayout, contentRow int) {
	total := len(m.session.Files)
	if total == 0 {
		return
	}
	start := fileListStart(m.selected, total, l.fileListItems.h)
	idx := start + contentRow
	if idx < 0 || idx >= total {
		return
	}
	if idx == m.selected {
		return
	}
	m.selected = idx
	m.cursor = 0
	m.scroll = 0
	m.selecting = false
}

func (m *model) clickDiff(rowOffset int) {
	rows := m.rows()
	if len(rows) == 0 {
		return
	}
	rowIdx := m.scroll + rowOffset
	if rowIdx < 0 || rowIdx >= len(rows) {
		return
	}
	m.cursor = rowIdx
	m.ensureCursorVisible()
}

func (m *model) scrollDiff(delta int) {
	rows := m.rows()
	if len(rows) == 0 {
		return
	}
	maxScroll := max(0, len(rows)-m.diffHeight())
	m.scroll = clamp(0, maxScroll, m.scroll+delta)
}
