package domain

import "sort"

type LineOp int

const (
	LineContext LineOp = iota
	LineDelete
	LineInsert
)

type Line struct {
	Op      LineOp
	OldNum  int
	NewNum  int
	Content string
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Section  string
	Lines    []Line
}

type FileChange struct {
	OldPath   string
	NewPath   string
	Hunks     []Hunk
	Binary    bool
	Untracked bool
	Staged    bool
}

func (f FileChange) Path() string {
	if f.NewPath != "" && f.NewPath != "/dev/null" {
		return f.NewPath
	}
	return f.OldPath
}

type DiffContext struct {
	Left              string
	Right             string
	IncludesStaged    bool
	IncludesUnstaged  bool
	IncludesUntracked bool
}

type CommentAnchor struct {
	File      string
	StartLine int
	EndLine   int
}

type Comment struct {
	Anchor CommentAnchor
	Text   string
}

type ReviewSession struct {
	Context        DiffContext
	Files          []FileChange
	Comments       []Comment
	GeneralComment string
	ReadFiles      map[string]bool
}

func NewReviewSession(ctx DiffContext, files []FileChange) ReviewSession {
	return ReviewSession{
		Context:   ctx,
		Files:     files,
		ReadFiles: make(map[string]bool),
	}
}

func (s *ReviewSession) AddComment(anchor CommentAnchor, text string) {
	if anchor.File == "" || anchor.StartLine < 1 || text == "" {
		return
	}
	if anchor.EndLine > 0 && anchor.EndLine < anchor.StartLine {
		anchor.StartLine, anchor.EndLine = anchor.EndLine, anchor.StartLine
	}
	s.Comments = append(s.Comments, Comment{Anchor: anchor, Text: text})
}

func (s *ReviewSession) DeleteComment(index int) bool {
	if index < 0 || index >= len(s.Comments) {
		return false
	}
	s.Comments = append(s.Comments[:index], s.Comments[index+1:]...)
	return true
}

func (s *ReviewSession) ToggleRead(path string) bool {
	if s.ReadFiles == nil {
		s.ReadFiles = make(map[string]bool)
	}
	if s.ReadFiles[path] {
		delete(s.ReadFiles, path)
		return false
	}
	s.ReadFiles[path] = true
	return true
}

func (s ReviewSession) FileByPath(path string) (FileChange, bool) {
	for _, f := range s.Files {
		if f.Path() == path {
			return f, true
		}
	}
	return FileChange{}, false
}

func (s ReviewSession) SortedComments() []Comment {
	out := append([]Comment(nil), s.Comments...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Anchor.File != out[j].Anchor.File {
			return out[i].Anchor.File < out[j].Anchor.File
		}
		if out[i].Anchor.StartLine != out[j].Anchor.StartLine {
			return out[i].Anchor.StartLine < out[j].Anchor.StartLine
		}
		return out[i].Text < out[j].Text
	})
	return out
}

type RepoStatus struct {
	Path      string
	Staged    bool
	Untracked bool
}

type Capabilities struct {
	CanStage           bool
	CanReadWorkingTree bool
	CanOpenEditor      bool
	CanPollStatus      bool
}
