package main

import (
	"go/token"
	"testing"

	"github.com/uudashr/gocognit"
)

func testStat(filename string, line int, name string) gocognit.Stat {
	return gocognit.Stat{
		FuncName: name,
		Pos:      token.Position{Filename: filename, Line: line, Column: 1},
	}
}

func TestDedupeStats(t *testing.T) {
	in := []gocognit.Stat{
		testStat("/p/a.go", 1, "A"),
		testStat("/p/a.go", 1, "A"), // duplicate from a test variant
		testStat("/p/a.go", 2, "B"),
		testStat("/p/b.go", 1, "A"), // same name, different position
	}

	got := dedupeStats(in)
	if len(got) != 3 {
		t.Fatalf("dedupeStats returned %d stats, want 3: %v", len(got), got)
	}
}

func TestStatFileWithinTarget(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		target   string
		isFile   bool
		want     bool
	}{
		{name: "dir direct child", filename: "/p/a.go", target: "/p", want: true},
		{name: "dir nested child", filename: "/p/sub/a.go", target: "/p", want: true},
		{name: "dir outside", filename: "/q/a.go", target: "/p", want: false},
		{name: "dir prefix is not membership", filename: "/px/a.go", target: "/p", want: false},
		{name: "file exact", filename: "/p/a.go", target: "/p/a.go", isFile: true, want: true},
		{name: "file other", filename: "/p/b.go", target: "/p/a.go", isFile: true, want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := statFileWithinTarget(tt.filename, tt.target, tt.isFile); got != tt.want {
				t.Errorf("statFileWithinTarget(%q, %q, %v) = %v, want %v", tt.filename, tt.target, tt.isFile, got, tt.want)
			}
		})
	}
}

func TestFilterStatFiles(t *testing.T) {
	stats := []gocognit.Stat{
		testStat("/p/a.go", 1, "A"),
		testStat("/p/sub/b.go", 1, "B"),
		testStat("/other/c.go", 1, "C"),
	}

	dir := filterStatFiles(stats, "/p", false)
	if len(dir) != 2 {
		t.Fatalf("dir filter kept %d stats, want 2: %v", len(dir), dir)
	}

	file := filterStatFiles(stats, "/p/a.go", true)
	if len(file) != 1 || file[0].FuncName != "A" {
		t.Fatalf("file filter = %v, want only A", file)
	}
}
