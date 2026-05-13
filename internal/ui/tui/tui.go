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
	ctx      context.Context
	app      reviewapp.App
	opts     Options
	session  domain.ReviewSession
	mode     mode
	view     viewMode
	lineNums lineNumMode
	width    int
	height   int
	selected int
	cursor   int
	scroll   int
	input    textarea.Model
	status   string
	saved    bool
	count    string
	search   string
}

type stageMsg struct {
	session domain.ReviewSession
	path    string
	err     error
}

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
	addStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	delStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hunkStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	commentStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
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

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	key := msg.String()
	if m.appendCount(key) {
		return m, nil
	}
	count, hasCount := m.consumeCount(1)
	switch key {
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
		m.moveCursor(count * m.pageStep())
	case "pgup", "ctrl+b":
		m.moveCursor(-count * m.pageStep())
	case "ctrl+d":
		m.moveCursor(count * m.halfPageStep())
	case "ctrl+u":
		m.moveCursor(-count * m.halfPageStep())
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

func (m model) View() string {
	if m.mode == modeHelp {
		return m.helpView()
	}
	if m.mode == modeComment || m.mode == modeGeneral || m.mode == modeSearch {
		title := "Inline Comment"
		if m.mode == modeGeneral {
			title = "General Comment"
		}
		if m.mode == modeSearch {
			title = "Search"
		}
		controls := "ctrl+s save  esc cancel"
		if m.mode == modeSearch {
			controls = "enter search  esc cancel"
		}
		return panelStyle.Width(max(20, m.width-2)).Render(
			headerStyle.Render(title) + "\n\n" +
				m.input.View() + "\n\n" +
				statusStyle.Render(controls),
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
	return "j/k move  h/l file  / search  42G line  J/K hunk  Tab view  Ctrl+n nums  c comment  s save  q quit  ? help"
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
		if i == m.cursor {
			line = selectedStyle.Width(rowW).Render(stripAnsi(line))
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

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

func padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) > width {
		if width == 1 {
			return string(rs[:1])
		}
		return string(rs[:width-1]) + "…"
	}
	if len(rs) < width {
		return s + strings.Repeat(" ", width-len(rs))
	}
	return s
}

func stripAnsi(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
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

func (m model) diffHeight() int {
	headerHeight := lipgloss.Height(m.header())
	bodyHeight := max(6, m.height-headerHeight-2)
	return max(1, bodyHeight-2)
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

func expandTabs(s string, tabStop int) string {
	if tabStop < 1 || !strings.ContainsRune(s, '\t') {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			spaces := tabStop - col%tabStop
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
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

func fileListStart(selected, total, height int) int {
	if total <= 0 || height <= 0 {
		return 0
	}
	selected = clamp(0, total-1, selected)
	if total <= height {
		return 0
	}
	if selected < height {
		return 0
	}
	return min(total-height, selected-height+1)
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

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func mod(a, b int) int {
	if b == 0 {
		return 0
	}
	r := a % b
	if r < 0 {
		return r + b
	}
	return r
}
