package tui

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
