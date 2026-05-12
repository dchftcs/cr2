package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dc/cr2/internal/domain"
	"github.com/dc/cr2/internal/export"
	"github.com/dc/cr2/internal/reviewapp"
)

type Options struct {
	OutputFile string
}

type UI struct {
	app  reviewapp.App
	spec reviewapp.DiffSpec
	opts Options
}

func New(app reviewapp.App, spec reviewapp.DiffSpec, opts Options) UI {
	return UI{app: app, spec: spec, opts: opts}
}

func (ui UI) Run(ctx context.Context) error {
	session, err := ui.app.Load(ctx, ui.spec)
	if err != nil {
		return err
	}
	m := newModel(ctx, ui.app, session, ui.opts)
	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	fm, ok := final.(model)
	if !ok || !fm.saved {
		return nil
	}
	md := export.Markdown(fm.session)
	if fm.opts.OutputFile == "" {
		fmt.Print(md)
		return nil
	}
	if err := os.WriteFile(fm.opts.OutputFile, []byte(md), 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Review saved to %s\n", fm.opts.OutputFile)
	return nil
}

type mode int

const (
	modeNormal mode = iota
	modeComment
	modeGeneral
	modeHelp
)

type model struct {
	ctx      context.Context
	app      reviewapp.App
	opts     Options
	session  domain.ReviewSession
	mode     mode
	width    int
	height   int
	selected int
	cursor   int
	scroll   int
	input    textarea.Model
	status   string
	saved    bool
}

type stageMsg struct {
	session domain.ReviewSession
	path    string
	err     error
}

type diffRow struct {
	text    string
	lineNum int
	op      domain.LineOp
	comment *domain.Comment
}

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
	addStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	delStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hunkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	commentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

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
	case tea.KeyMsg:
		switch m.mode {
		case modeComment:
			return m.updateComment(msg)
		case modeGeneral:
			return m.updateGeneral(msg)
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

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "s":
		m.saved = true
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "pgdown", "ctrl+f":
		m.moveCursor(max(1, m.diffHeight()-2))
	case "pgup", "ctrl+b":
		m.moveCursor(-max(1, m.diffHeight()-2))
	case "G":
		m.cursor = max(0, len(m.rows())-1)
		m.ensureCursorVisible()
	case "g":
		m.cursor = 0
		m.ensureCursorVisible()
	case "]", "right":
		m.moveFile(1)
	case "[", "left":
		m.moveFile(-1)
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
			line := m.currentLine()
			if line < 1 {
				m.status = "Select a source line before commenting"
				return m, nil
			}
			m.input = textarea.New()
			m.input.Placeholder = fmt.Sprintf("Comment on %s:%d", f.Path(), line)
			m.input.ShowLineNumbers = false
			m.input.SetWidth(max(20, m.width-4))
			m.input.SetHeight(6)
			m.mode = modeComment
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

func (m model) updateComment(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "ctrl+s":
		text := strings.TrimSpace(m.input.Value())
		if text != "" {
			if f, ok := m.currentFile(); ok {
				m.session.AddComment(domain.CommentAnchor{
					File:      f.Path(),
					StartLine: m.currentLine(),
				}, text)
				m.status = "Comment added"
			}
		}
		m.mode = modeNormal
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

func (m model) toggleStageCmd(path string) tea.Cmd {
	session := m.session
	app := m.app
	ctx := m.ctx
	return func() tea.Msg {
		err := app.ToggleStage(ctx, &session, path)
		return stageMsg{session: session, path: path, err: err}
	}
}

func (m model) View() string {
	if m.mode == modeHelp {
		return m.helpView()
	}
	if m.mode == modeComment || m.mode == modeGeneral {
		title := "Inline Comment"
		if m.mode == modeGeneral {
			title = "General Comment"
		}
		return panelStyle.Width(max(20, m.width-2)).Render(
			headerStyle.Render(title) + "\n\n" +
				m.input.View() + "\n\n" +
				statusStyle.Render("ctrl+s save  esc cancel"),
		)
	}

	header := m.header()
	bodyHeight := max(6, m.height-lipgloss.Height(header)-2)
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
	return "j/k move  ]/[ file  c comment  R general  m read  a stage  s save  ? help  q quit"
}

func (m model) fileListView(width, height int) string {
	if len(m.session.Files) == 0 {
		return "No changes."
	}
	var lines []string
	for i, f := range m.session.Files {
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
	return strings.Join(limit(lines, height), "\n")
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
	var out []string
	out = append(out, headerStyle.Render(truncate(f.Path(), width-2)))
	for i := start; i < end; i++ {
		row := rows[i]
		line := truncate(row.text, width-2)
		switch {
		case row.comment != nil:
			line = commentStyle.Render(line)
		case strings.HasPrefix(row.text, "@@"):
			line = hunkStyle.Render(line)
		case row.op == domain.LineInsert:
			line = addStyle.Render(line)
		case row.op == domain.LineDelete:
			line = delStyle.Render(line)
		}
		if i == m.cursor {
			line = selectedStyle.Width(max(1, width-2)).Render(line)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func (m model) helpView() string {
	return panelStyle.Width(max(20, m.width-2)).Render(`cr keys

j/k, up/down       move in diff
PgUp/PgDn          page diff
g/G                top/bottom
]/[, left/right    next/previous file
c                  add inline comment at selected line
R                  edit general review comment
ctrl+s             submit comment while editing
m                  mark selected file read/unread
a                  stage/unstage selected file
s                  save review and exit
q                  quit without saving
?                  close help`)
}

func (m *model) moveFile(delta int) {
	if len(m.session.Files) == 0 {
		return
	}
	m.selected = clamp(0, len(m.session.Files)-1, m.selected+delta)
	m.cursor = 0
	m.scroll = 0
}

func (m *model) moveCursor(delta int) {
	rows := m.rows()
	if len(rows) == 0 {
		return
	}
	m.cursor = clamp(0, len(rows)-1, m.cursor+delta)
	m.ensureCursorVisible()
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

func (m model) diffHeight() int {
	return max(1, m.height-5)
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

func (m model) rows() []diffRow {
	f, ok := m.currentFile()
	if !ok {
		return nil
	}
	comments := commentsByLine(m.session.Comments, f.Path())
	var rows []diffRow
	for _, h := range f.Hunks {
		rows = append(rows, diffRow{
			text: fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", h.OldStart, h.OldCount, h.NewStart, h.NewCount, h.Section),
		})
		for _, line := range h.Lines {
			prefix := " "
			num := line.NewNum
			if line.Op == domain.LineDelete {
				prefix = "-"
				num = line.OldNum
			}
			if line.Op == domain.LineInsert {
				prefix = "+"
			}
			rows = append(rows, diffRow{
				text:    fmt.Sprintf("%s%5d %s", prefix, num, line.Content),
				lineNum: num,
				op:      line.Op,
			})
			if line.Op != domain.LineDelete {
				for _, comment := range comments[num] {
					c := comment
					rows = append(rows, diffRow{
						text:    "      > " + firstLine(comment.Text),
						lineNum: num,
						comment: &c,
					})
				}
			}
		}
	}
	return rows
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

func replaceAt(s string, idx int, r rune) string {
	rs := []rune(s)
	if idx >= 0 && idx < len(rs) {
		rs[idx] = r
	}
	return string(rs)
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= width {
		return s
	}
	if width == 1 {
		return string(rs[:1])
	}
	return string(rs[:width-1]) + "…"
}

func limit(lines []string, n int) []string {
	if n < 0 {
		return nil
	}
	if len(lines) <= n {
		return lines
	}
	return lines[:n]
}

func clamp(lo, hi, v int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
