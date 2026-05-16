package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dc/cr2/internal/domain"
)

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	key := msg.String()
	if m.appendCount(key) {
		return m, nil
	}
	count, hasCount := m.consumeCount(1)
	switch key {
	case "esc":
		if m.selecting {
			m.selecting = false
			m.status = "Selection cleared"
		}
	case "q", "ctrl+c":
		return m, tea.Quit
	case "s":
		m.saved = true
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "/":
		m.input = textarea.New()
		m.input.Placeholder = "Search diff"
		m.input.ShowLineNumbers = false
		m.input.SetWidth(max(20, m.width-4))
		m.input.SetHeight(1)
		m.input.SetValue(m.search)
		m.mode = modeSearch
		return m, m.input.Focus()
	case "n":
		m.repeatSearch(count)
	case "N":
		m.repeatSearch(-count)
	case "j", "down":
		m.moveCursor(count)
	case "k", "up":
		m.moveCursor(-count)
	case "pgdown", "ctrl+f", " ":
		m.scrollDiff(count * m.pageStep())
	case "pgup", "ctrl+b":
		m.scrollDiff(-count * m.pageStep())
	case "ctrl+d":
		m.scrollDiff(count * m.halfPageStep())
	case "ctrl+u":
		m.scrollDiff(-count * m.halfPageStep())
	case "G":
		if hasCount {
			m.jumpToLine(count)
		} else {
			m.jumpBottom()
		}
	case "g", "home":
		if hasCount {
			m.jumpToLine(count)
		} else {
			m.jumpTop()
		}
	case "end":
		m.jumpBottom()
	case "J", "}":
		m.moveHunk(count)
	case "K", "{":
		m.moveHunk(-count)
	case "]", "right", "l":
		m.moveFile(count)
	case "[", "left", "h":
		m.moveFile(-count)
	case "tab":
		line := m.currentLine()
		if m.view == viewSideBySide {
			m.view = viewUnified
		} else {
			m.view = viewSideBySide
		}
		m.cursorToLine(line)
	case "ctrl+n":
		m.cycleLineNumMode()
	case "v":
		if line, ok := m.currentSourceLine(); ok {
			if m.selecting {
				m.selecting = false
				m.status = "Selection cleared"
			} else {
				m.selecting = true
				m.selectAnchor = line
				m.status = fmt.Sprintf("Selection started at line %d", line)
			}
		} else {
			m.status = "Select a source line before starting a range"
		}
	case "m":
		if f, ok := m.currentFile(); ok {
			read := m.session.ToggleRead(f.Path())
			m.status = fmt.Sprintf("%s read=%v", f.Path(), read)
		}
	case "a":
		if f, ok := m.currentFile(); ok {
			return m, m.toggleStageCmd(f.Path())
		}
	case "c":
		if f, ok := m.currentFile(); ok {
			anchor, ok := m.currentCommentAnchor(f.Path())
			if !ok {
				m.status = "Select a source line before commenting"
				return m, nil
			}
			m.input = textarea.New()
			m.input.Placeholder = "Write review comment..."
			m.input.ShowLineNumbers = false
			m.input.SetWidth(max(20, m.width-4))
			m.input.SetHeight(6)
			m.commentAnchor = anchor
			m.mode = modeComment
			m.ensureInlineEditorVisible()
			return m, m.input.Focus()
		}
	case "R":
		m.input = textarea.New()
		m.input.Placeholder = "General review comment"
		m.input.ShowLineNumbers = false
		m.input.SetWidth(max(20, m.width-4))
		m.input.SetHeight(8)
		m.input.SetValue(m.session.GeneralComment)
		m.mode = modeGeneral
		return m, m.input.Focus()
	}
	return m, nil
}

func (m *model) appendCount(key string) bool {
	if len(key) != 1 || key[0] < '0' || key[0] > '9' {
		return false
	}
	if key == "0" && m.count == "" {
		return false
	}
	if len(m.count) < 6 {
		m.count += key
	}
	m.status = "count: " + m.count
	return true
}

func (m *model) consumeCount(defaultValue int) (int, bool) {
	if m.count == "" {
		return defaultValue, false
	}
	n := 0
	for _, r := range m.count {
		n = n*10 + int(r-'0')
	}
	m.count = ""
	if n < 1 {
		return defaultValue, false
	}
	return n, true
}

func (m model) updateComment(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.commentAnchor = domain.CommentAnchor{}
		return m, nil
	case "ctrl+s":
		text := strings.TrimSpace(m.input.Value())
		if text != "" {
			if m.commentAnchor.File != "" {
				m.session.AddComment(m.commentAnchor, text)
				m.status = "Comment added"
			}
		}
		m.mode = modeNormal
		m.selecting = false
		m.commentAnchor = domain.CommentAnchor{}
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m model) updateGeneral(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "ctrl+s":
		m.session.GeneralComment = strings.TrimRight(m.input.Value(), "\n")
		m.status = "General comment saved"
		m.mode = modeNormal
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "enter", "ctrl+s":
		query := strings.TrimSpace(m.input.Value())
		m.mode = modeNormal
		if query == "" {
			return m, nil
		}
		m.search = query
		if !m.findNextSearchMatch(1) {
			m.status = "No matches for " + query
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m model) toggleStageCmd(path string) tea.Cmd {
	session := m.session
	app := m.app
	ctx := m.ctx
	return func() tea.Msg {
		err := app.ToggleStage(ctx, &session, path)
		return stageMsg{session: session, path: path, err: err}
	}
}
