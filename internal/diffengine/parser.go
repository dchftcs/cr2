package diffengine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dc/cr2/internal/domain"
)

func ParseUnified(raw string) ([]domain.FileChange, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var files []domain.FileChange
	var cur *domain.FileChange
	var hunk *domain.Hunk
	var oldNum, newNum int

	flushFile := func() {
		if cur != nil {
			files = append(files, *cur)
		}
		cur = nil
		hunk = nil
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flushFile()
			cur = &domain.FileChange{}
		case strings.HasPrefix(line, "rename from "):
			if cur == nil {
				cur = &domain.FileChange{}
			}
			cur.OldPath = strings.TrimSpace(strings.TrimPrefix(line, "rename from "))
			cur.Renamed = true
		case strings.HasPrefix(line, "rename to "):
			if cur == nil {
				cur = &domain.FileChange{}
			}
			cur.NewPath = strings.TrimSpace(strings.TrimPrefix(line, "rename to "))
			cur.Renamed = true
		case strings.HasPrefix(line, "Binary files "):
			if cur != nil {
				cur.Binary = true
			}
		case strings.HasPrefix(line, "--- "):
			if cur == nil {
				cur = &domain.FileChange{}
			}
			cur.OldPath = cleanDiffPath(strings.TrimSpace(strings.TrimPrefix(line, "--- ")))
		case strings.HasPrefix(line, "+++ "):
			if cur == nil {
				cur = &domain.FileChange{}
			}
			cur.NewPath = cleanDiffPath(strings.TrimSpace(strings.TrimPrefix(line, "+++ ")))
			if cur.OldPath == "/dev/null" {
				cur.Untracked = true
			}
		case strings.HasPrefix(line, "@@ "):
			if cur == nil {
				return nil, fmt.Errorf("hunk found before file header")
			}
			parsed, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			cur.Hunks = append(cur.Hunks, parsed)
			hunk = &cur.Hunks[len(cur.Hunks)-1]
			oldNum = hunk.OldStart
			newNum = hunk.NewStart
		case hunk != nil && len(line) > 0:
			switch line[0] {
			case ' ':
				hunk.Lines = append(hunk.Lines, domain.Line{
					Op:      domain.LineContext,
					OldNum:  oldNum,
					NewNum:  newNum,
					Content: line[1:],
				})
				oldNum++
				newNum++
			case '-':
				hunk.Lines = append(hunk.Lines, domain.Line{
					Op:      domain.LineDelete,
					OldNum:  oldNum,
					Content: line[1:],
				})
				oldNum++
			case '+':
				hunk.Lines = append(hunk.Lines, domain.Line{
					Op:      domain.LineInsert,
					NewNum:  newNum,
					Content: line[1:],
				})
				newNum++
			}
		}
	}
	flushFile()
	return files, nil
}

func cleanDiffPath(p string) string {
	p = strings.TrimSpace(p)
	if tab := strings.IndexByte(p, '\t'); tab >= 0 {
		p = p[:tab]
	}
	p = strings.TrimPrefix(p, "a/")
	p = strings.TrimPrefix(p, "b/")
	return p
}

func parseHunkHeader(line string) (domain.Hunk, error) {
	end := strings.Index(line[3:], " @@")
	if end < 0 {
		return domain.Hunk{}, fmt.Errorf("invalid hunk header %q", line)
	}
	spec := line[3 : 3+end]
	section := strings.TrimSpace(line[3+end+3:])
	parts := strings.Fields(spec)
	if len(parts) != 2 {
		return domain.Hunk{}, fmt.Errorf("invalid hunk range %q", line)
	}
	oldStart, oldCount, err := parseRange(parts[0], '-')
	if err != nil {
		return domain.Hunk{}, err
	}
	newStart, newCount, err := parseRange(parts[1], '+')
	if err != nil {
		return domain.Hunk{}, err
	}
	return domain.Hunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
		Section:  section,
	}, nil
}

func parseRange(v string, prefix byte) (int, int, error) {
	if len(v) < 2 || v[0] != prefix {
		return 0, 0, fmt.Errorf("invalid hunk range %q", v)
	}
	body := v[1:]
	count := 1
	if comma := strings.IndexByte(body, ','); comma >= 0 {
		var err error
		count, err = strconv.Atoi(body[comma+1:])
		if err != nil {
			return 0, 0, err
		}
		body = body[:comma]
	}
	start, err := strconv.Atoi(body)
	if err != nil {
		return 0, 0, err
	}
	return start, count, nil
}
