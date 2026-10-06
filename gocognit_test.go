package gocognit_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/uudashr/gocognit"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, gocognit.Analyzer, "a")
}

func TestAnalyzerOver3(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "3")
	analysistest.Run(t, testdata, gocognit.Analyzer, "b")
}

func TestAnalyzerGenerics(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, gocognit.Analyzer, "c")
}

func TestAnalyzerComplex(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, gocognit.Analyzer, "d")
}

func TestAnalyzerLogicalOp(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, gocognit.Analyzer, "e")
}

func TestAnalyzerRecursion(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, gocognit.Analyzer, "f")
}

// TestComplexityDirectRecursion pins the single-function API (Complexity /
// ScanComplexity). It has no package context, so only direct self-calls are
// counted; direct method and generic self-calls are covered by the shared
// resolver, but indirect cycles are out of reach.
func TestComplexityDirectRecursion(t *testing.T) {
	fset := token.NewFileSet()
	byName := make(map[string]*ast.FuncDecl)

	for _, name := range []string{"testdata/src/f/f.go", "testdata/src/f/f2.go"} {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}

		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				byName[fn.Name.Name] = fn
			}
		}
	}

	want := map[string]int{
		"Fib":              2, // if + direct recursion
		"Sum":              2, // if + generic direct recursion
		"Loop":             1, // direct method self-call
		"MutualA":          1, // indirect only: not visible without the package
		"MutualB":          0,
		"Walk":             1, // only the for; mutual method recursion is indirect
		"Step":             0,
		"notRecursive":     1,
		"CallerCallsCycle": 1, // calls into a cycle but is not recursive itself
		"helper":           0,
	}

	for name, expected := range want {
		fn, ok := byName[name]
		if !ok {
			t.Fatalf("function %s not found", name)
		}

		if got := gocognit.Complexity(fn); got != expected {
			t.Errorf("Complexity(%s) = %d, want %d", name, got, expected)
		}
	}
}

// TestRecursiveFuncsSyntactic pins the behavior of the type-less fallback used
// by the command line tool. It resolves package-level functions across files
// and direct receiver calls, but (by design) cannot see mutual method
// recursion, which the analyzer resolves with type information.
func TestRecursiveFuncsSyntactic(t *testing.T) {
	fset := token.NewFileSet()

	var files []*ast.File
	for _, name := range []string{"testdata/src/f/f.go", "testdata/src/f/f2.go"} {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}

		files = append(files, f)
	}

	got := gocognit.RecursiveFuncs(files, nil)

	want := map[string]bool{
		"Fib":              true,
		"MutualA":          true,
		"MutualB":          true,
		"Cycle1":           true,
		"Cycle2":           true,
		"Cycle3":           true,
		"Sum":              true,  // generic instantiation is resolved with types
		"CallerCallsCycle": false, // calls into a cycle without being in one
		"helper":           false,
		"notRecursive":     false,
		"Walk":             false, // mutual method calls need type information
		"Step":             false,
		"Loop":             true, // direct receiver self-call is recognized
	}

	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if got[fn] != want[fn.Name.Name] {
				t.Errorf("RecursiveFuncs(%s) = %v, want %v", fn.Name.Name, got[fn], want[fn.Name.Name])
			}
		}
	}
}
