package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dc/cr2/internal/domain"
	"github.com/dc/cr2/internal/reviewapp"
)

type mode int

const (
	modeNormal mode = iota
	modeComment
	modeGeneral
	modeSearch
	modeHelp
)

type viewMode int

const (
	viewSideBySide viewMode = iota
	viewUnified
)

type lineNumMode int

const (
	lineNumBoth lineNumMode = iota
	lineNumRelativeOnly
	lineNumAbsoluteOnly
)

type model struct {
	ctx           context.Context
	app           reviewapp.App
	opts          Options
	session       domain.ReviewSession
	mode          mode
	view          viewMode
	lineNums      lineNumMode
	width         int
	height        int
	selected      int
	cursor        int
	scroll        int
	input         textarea.Model
	status        string
	saved         bool
	count         string
	search        string
	selecting     bool
	selectAnchor  int
	commentAnchor domain.CommentAnchor
}

type stageMsg struct {
	session domain.ReviewSession
	path    string
	err     error
}

func newModel(ctx context.Context, app reviewapp.App, session domain.ReviewSession, opts Options) model {
	input := textarea.New()
	input.Placeholder = "Write review comment..."
	input.SetWidth(80)
	input.SetHeight(6)
	input.ShowLineNumbers = false
	return model{
		ctx:     ctx,
		app:     app,
		opts:    opts,
		session: session,
		input:   input,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(max(20, msg.Width-4))
		return m, nil
	case stageMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.session = msg.session
		m.status = "Toggled stage state for " + msg.path
		return m, nil
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tea.KeyMsg:
		switch m.mode {
		case modeComment:
			return m.updateComment(msg)
		case modeGeneral:
			return m.updateGeneral(msg)
		case modeSearch:
			return m.updateSearch(msg)
		case modeHelp:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "?" {
				m.mode = modeNormal
			}
			return m, nil
		default:
			return m.updateNormal(msg)
		}
	}
	return m, nil
}

func (m *model) cycleLineNumMode() {
	switch m.lineNums {
	case lineNumBoth:
		m.lineNums = lineNumRelativeOnly
		m.status = "Line numbers: relative"
	case lineNumRelativeOnly:
		m.lineNums = lineNumAbsoluteOnly
		m.status = "Line numbers: absolute"
	case lineNumAbsoluteOnly:
		m.lineNums = lineNumBoth
		m.status = "Line numbers: relative + absolute"
	}
}

func (m model) showRelativeLineNums() bool {
	return m.lineNums == lineNumBoth || m.lineNums == lineNumRelativeOnly
}

func (m model) showAbsoluteLineNums() bool {
	return m.lineNums == lineNumBoth || m.lineNums == lineNumAbsoluteOnly
}
