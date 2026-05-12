package export

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dc/cr2/internal/domain"
)

func Markdown(session domain.ReviewSession) string {
	var b strings.Builder
	b.WriteString("# Code Review\n\n")
	writeContext(&b, session.Context)

	files := make(map[string]domain.FileChange, len(session.Files))
	for _, f := range session.Files {
		files[f.Path()] = f
	}

	byFile := make(map[string][]domain.Comment)
	for _, c := range session.SortedComments() {
		byFile[c.Anchor.File] = append(byFile[c.Anchor.File], c)
	}
	var names []string
	for name := range byFile {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		b.WriteString(fmt.Sprintf("## File: `%s`\n\n", name))
		for _, c := range byFile[name] {
			b.WriteString(fmt.Sprintf("### `%s`\n", location(c.Anchor)))
			if snippet := snippet(files[name], c.Anchor); snippet != "" {
				b.WriteString("```")
				if ext := extension(name); ext != "" {
					b.WriteString(ext)
				}
				b.WriteString("\n")
				b.WriteString(snippet)
				b.WriteString("\n```\n")
			}
			b.WriteString("**Comment:**\n")
			for _, line := range strings.Split(c.Text, "\n") {
				b.WriteString("> ")
				b.WriteString(line)
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
	}

	if strings.TrimSpace(session.GeneralComment) != "" {
		b.WriteString("## General Comments\n\n")
		b.WriteString(strings.TrimRight(session.GeneralComment, "\n"))
		b.WriteString("\n")
	}
	if len(session.Comments) == 0 && strings.TrimSpace(session.GeneralComment) == "" {
		b.WriteString("No comments.\n")
	}
	return b.String()
}

func writeContext(b *strings.Builder, ctx domain.DiffContext) {
	if ctx.Left == "" && ctx.Right == "" {
		return
	}
	b.WriteString("## Diff Context\n\n")
	if ctx.Left != "" {
		b.WriteString(fmt.Sprintf("- LHS (base): `%s`\n", ctx.Left))
	}
	if ctx.Right != "" {
		b.WriteString(fmt.Sprintf("- RHS (target): `%s`\n", ctx.Right))
	}
	b.WriteString(fmt.Sprintf("- Includes staged changes on RHS: %s\n", yesNo(ctx.IncludesStaged)))
	b.WriteString(fmt.Sprintf("- Includes unstaged changes on RHS: %s\n", yesNo(ctx.IncludesUnstaged)))
	b.WriteString(fmt.Sprintf("- Includes untracked files on RHS: %s\n", yesNo(ctx.IncludesUntracked)))
	b.WriteString("- Inline comment locations are relative to this diff context.\n\n")
}

func location(a domain.CommentAnchor) string {
	if a.EndLine > 0 && a.EndLine != a.StartLine {
		return fmt.Sprintf("%s:%d-%d", a.File, a.StartLine, a.EndLine)
	}
	return fmt.Sprintf("%s:%d", a.File, a.StartLine)
}

func snippet(file domain.FileChange, anchor domain.CommentAnchor) string {
	if file.Path() == "" {
		return ""
	}
	start := anchor.StartLine - 2
	end := anchor.StartLine + 2
	if anchor.EndLine > 0 {
		start = anchor.StartLine - 1
		end = anchor.EndLine + 1
	}
	var rows []string
	for _, h := range file.Hunks {
		for _, line := range h.Lines {
			num := line.NewNum
			if num == 0 {
				num = line.OldNum
			}
			if num >= start && num <= end {
				rows = append(rows, fmt.Sprintf("%d\t%s", num, line.Content))
			}
		}
	}
	return strings.Join(rows, "\n")
}

func extension(name string) string {
	idx := strings.LastIndexByte(name, '.')
	if idx < 0 || idx == len(name)-1 {
		return ""
	}
	return name[idx+1:]
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
