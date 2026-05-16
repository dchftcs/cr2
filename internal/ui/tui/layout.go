package tui

import "github.com/charmbracelet/lipgloss"

// Layout dimensions. Keep these constants here so both rendering and
// hit-testing agree on panel borders, minimum sizes, and the footer height.
const (
	panelBorderWidth          = 1
	footerHeight              = 1
	minPanelContentHeight     = 6
	minLeftPanelContentWidth  = 28
	maxLeftPanelContentWidth  = 44
	minRightPanelContentWidth = 20
)

// rect is a half-open rectangle in terminal cells. A point (x, y) is inside
// when r.x <= x < r.x+r.w && r.y <= y < r.y+r.h.
type rect struct {
	x, y, w, h int
}

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

func (r rect) inset(top, right, bottom, left int) rect {
	return rect{
		x: r.x + left,
		y: r.y + top,
		w: max(0, r.w-left-right),
		h: max(0, r.h-top-bottom),
	}
}

// viewLayout is the bounding box of every interactive region of the workspace
// view. Rendering sizes panels from these rects, and the mouse handler hit-tests
// against them — so the two views of the screen cannot drift apart.
//
// Rectangles are computed for normal-mode rendering (the only mode that
// dispatches mouse events). Comment-overlay modes that consume extra space are
// handled inline in their own render paths.
type viewLayout struct {
	workspace     rect
	header        rect
	fileListPanel rect // includes panel border
	fileListItems rect // scrollable list area inside the panel
	diffPanel     rect // includes panel border
	diffContent   rect // full inner area of the diff panel (diffHeader ∪ diffRows + any padding)
	diffHeader    rect // file path + optional rename subheader (non-scrolling)
	diffRows      rect // scrollable diff rows
	footer        rect
}

func (m model) computeLayout() viewLayout {
	headerH := lipgloss.Height(m.header())

	workspace := rect{x: 0, y: 0, w: m.width, h: m.height}
	header := rect{x: 0, y: 0, w: m.width, h: headerH}
	footer := rect{
		x: 0,
		y: max(0, m.height-footerHeight),
		w: m.width,
		h: footerHeight,
	}

	panelsTop := headerH
	panelsH := max(
		minPanelContentHeight+2*panelBorderWidth,
		m.height-headerH-footerHeight,
	)

	leftContentW := clamp(minLeftPanelContentWidth, maxLeftPanelContentWidth, max(minLeftPanelContentWidth, m.width/3))
	leftPanelW := leftContentW + 2*panelBorderWidth
	rightPanelW := max(minRightPanelContentWidth+2*panelBorderWidth, m.width-leftPanelW)

	fileListPanel := rect{x: 0, y: panelsTop, w: leftPanelW, h: panelsH}
	diffPanel := rect{x: leftPanelW, y: panelsTop, w: rightPanelW, h: panelsH}

	fileListItems := fileListPanel.inset(panelBorderWidth, panelBorderWidth, panelBorderWidth, panelBorderWidth)
	diffContent := diffPanel.inset(panelBorderWidth, panelBorderWidth, panelBorderWidth, panelBorderWidth)

	diffHeaderH := 1
	if f, ok := m.currentFile(); ok && f.Renamed && m.view == viewSideBySide {
		diffHeaderH = 2
	}
	diffHeaderH = min(diffContent.h, diffHeaderH)

	diffHeader := rect{
		x: diffContent.x,
		y: diffContent.y,
		w: diffContent.w,
		h: diffHeaderH,
	}
	// diffView always reserves panelContent.h - 2 rows for scrolling content
	// (one row is used by the file header; the rename subheader, when present,
	// is anchored at row 1 of the content area).
	diffRows := rect{
		x: diffContent.x,
		y: diffContent.y + diffHeaderH,
		w: diffContent.w,
		h: max(0, diffContent.h-2),
	}

	return viewLayout{
		workspace:     workspace,
		header:        header,
		fileListPanel: fileListPanel,
		fileListItems: fileListItems,
		diffPanel:     diffPanel,
		diffContent:   diffContent,
		diffHeader:    diffHeader,
		diffRows:      diffRows,
		footer:        footer,
	}
}
