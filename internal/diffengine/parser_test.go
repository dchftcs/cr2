package diffengine

import (
	"testing"

	"github.com/dc/cr2/internal/domain"
)

func TestParseUnified(t *testing.T) {
	raw := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,2 +1,3 @@ func main
 package main
-old
+new
+extra
`
	files, err := ParseUnified(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d", len(files))
	}
	f := files[0]
	if f.Path() != "a.go" {
		t.Fatalf("path = %q", f.Path())
	}
	if got := len(f.Hunks[0].Lines); got != 4 {
		t.Fatalf("lines = %d", got)
	}
	if f.Hunks[0].Lines[1].Op != domain.LineDelete {
		t.Fatalf("line 2 op = %v", f.Hunks[0].Lines[1].Op)
	}
	if f.Hunks[0].Lines[2].NewNum != 2 {
		t.Fatalf("insert line num = %d", f.Hunks[0].Lines[2].NewNum)
	}
}

func TestParseRenameWithContentChanges(t *testing.T) {
	raw := `diff --git a/old.go b/new.go
similarity index 80%
rename from old.go
rename to new.go
index abc..def 100644
--- a/old.go
+++ b/new.go
@@ -1,2 +1,2 @@
 package main
-old
+new
`
	files, err := ParseUnified(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d", len(files))
	}
	f := files[0]
	if !f.Renamed {
		t.Fatal("expected Renamed=true")
	}
	if f.OldPath != "old.go" || f.NewPath != "new.go" {
		t.Fatalf("paths = %q -> %q, want old.go -> new.go", f.OldPath, f.NewPath)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("hunks = %d, want 1", len(f.Hunks))
	}
}

func TestParsePureRenameWithoutHunks(t *testing.T) {
	raw := `diff --git a/old.go b/new.go
similarity index 100%
rename from old.go
rename to new.go
`
	files, err := ParseUnified(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d", len(files))
	}
	f := files[0]
	if !f.Renamed {
		t.Fatal("expected Renamed=true")
	}
	if f.OldPath != "old.go" || f.NewPath != "new.go" {
		t.Fatalf("paths = %q -> %q, want old.go -> new.go", f.OldPath, f.NewPath)
	}
	if len(f.Hunks) != 0 {
		t.Fatalf("hunks = %d, want 0", len(f.Hunks))
	}
	if f.Path() != "new.go" {
		t.Fatalf("Path() = %q, want new.go", f.Path())
	}
}

func TestParseUntrackedNoIndexDiff(t *testing.T) {
	raw := `diff --git a/dev/null b/new.txt
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,2 @@
+one
+two
`
	files, err := ParseUnified(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d", len(files))
	}
	if !files[0].Untracked {
		t.Fatal("expected untracked file")
	}
	if files[0].Path() != "new.txt" {
		t.Fatalf("path = %q", files[0].Path())
	}
}
