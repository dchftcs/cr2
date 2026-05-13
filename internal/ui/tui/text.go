package tui

import "strings"

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
