package domain

import "testing"

func TestReviewSessionCommentsAndReadState(t *testing.T) {
	s := NewReviewSession(DiffContext{}, []FileChange{{NewPath: "a.go"}})
	s.AddComment(CommentAnchor{File: "a.go", StartLine: 10}, "fix this")
	if len(s.Comments) != 1 {
		t.Fatalf("comments = %d", len(s.Comments))
	}
	if read := s.ToggleRead("a.go"); !read {
		t.Fatal("expected file to become read")
	}
	if read := s.ToggleRead("a.go"); read {
		t.Fatal("expected file to become unread")
	}
	if !s.DeleteComment(0) {
		t.Fatal("delete comment failed")
	}
	if len(s.Comments) != 0 {
		t.Fatalf("comments = %d", len(s.Comments))
	}
}
