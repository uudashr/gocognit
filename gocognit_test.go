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

func TestAnalyzerIgnoreErrorChecks(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	gocognit.Analyzer.Flags.Set("ignore-error-checks", "true")
	t.Cleanup(func() {
		gocognit.Analyzer.Flags.Set("ignore-error-checks", "false")
	})
	analysistest.Run(t, testdata, gocognit.Analyzer, "g")
}

func TestAnalyzerIgnoreErrShorthand(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	gocognit.Analyzer.Flags.Set("ignore-err", "true")
	t.Cleanup(func() {
		gocognit.Analyzer.Flags.Set("ignore-err", "false")
	})
	analysistest.Run(t, testdata, gocognit.Analyzer, "g")
}

func TestComplexityIgnoreErrorChecks(t *testing.T) {
	src := `package test

func SimpleErrCheck(err error) error {
	if err != nil {
		return err
	}
	return nil
}

func ErrCheckWithInit() error {
	if err := doStep(); err != nil {
		return err
	}
	return nil
}

func NilNotEqualsErr(err error) error {
	if nil != err {
		return err
	}
	return nil
}

func CustomErrName(customErr error) error {
	if customErr != nil {
		return customErr
	}
	return nil
}

func SnakeCaseErrName(parse_err error) error {
	if parse_err != nil {
		return parse_err
	}
	return nil
}

func NestedErrCheck(err error) error {
	for i := 0; i < 10; i++ {
		if err != nil {
			return err
		}
	}
	return nil
}

func MultipleErrChecks(err, lastErr error) error {
	if err != nil {
		return err
	}
	if err := doStep(); err != nil {
		return err
	}
	if lastErr != nil {
		return lastErr
	}
	return nil
}

func NonErrCheck(n int) string {
	if n == 100 {
		return "a hundred"
	}
	return "others"
}

func CompoundErrCheck(err error, retry bool) error {
	if err != nil && retry {
		return err
	}
	return nil
}

func SuccessCheck(err error) error {
	if err == nil {
		return nil
	}
	return err
}

func ErrCheckWithElse(err error) error {
	if err != nil {
		return err
	} else {
		return nil
	}
}

func ElseIfErrCheck(cond bool, err error) error {
	if cond {
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func ErrCheckWithParen(err error) error {
	if (err != nil) {
		return err
	}
	return nil
}

func NestedBranchInsideErrCheck(err error, canRetry bool) error {
	if err != nil {
		if canRetry {
			return err
		}
		return err
	}
	return nil
}

func doStep() error {
	return nil
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	funcs := make(map[string]*ast.FuncDecl)
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcs[fn.Name.Name] = fn
		}
	}

	tests := []struct {
		name        string
		wantDefault int
		wantIgnored int
	}{
		{name: "SimpleErrCheck", wantDefault: 1, wantIgnored: 0},
		{name: "ErrCheckWithInit", wantDefault: 1, wantIgnored: 0},
		{name: "NilNotEqualsErr", wantDefault: 1, wantIgnored: 0},
		{name: "CustomErrName", wantDefault: 1, wantIgnored: 0},
		{name: "SnakeCaseErrName", wantDefault: 1, wantIgnored: 0},
		{name: "NestedErrCheck", wantDefault: 3, wantIgnored: 1},
		{name: "MultipleErrChecks", wantDefault: 3, wantIgnored: 0},
		{name: "NonErrCheck", wantDefault: 1, wantIgnored: 1},
		{name: "CompoundErrCheck", wantDefault: 2, wantIgnored: 2},
		{name: "SuccessCheck", wantDefault: 1, wantIgnored: 1},
		{name: "ErrCheckWithElse", wantDefault: 2, wantIgnored: 2},
		{name: "ElseIfErrCheck", wantDefault: 2, wantIgnored: 2},
		{name: "ErrCheckWithParen", wantDefault: 1, wantIgnored: 0},
		{name: "NestedBranchInsideErrCheck", wantDefault: 3, wantIgnored: 2},
	}

	for _, tt := range tests {
		fn, ok := funcs[tt.name]
		if !ok {
			t.Fatalf("function %s not found", tt.name)
		}

		gotDefault := gocognit.Complexity(fn)
		if gotDefault != tt.wantDefault {
			t.Errorf("%s default complexity = %d, want %d", tt.name, gotDefault, tt.wantDefault)
		}

		gotIgnored := gocognit.ComplexityWithOptions(fn, gocognit.ComplexityOptions{
			IgnoreErrorChecks: true,
		})
		if gotIgnored != tt.wantIgnored {
			t.Errorf("%s with IgnoreErrorChecks complexity = %d, want %d", tt.name, gotIgnored, tt.wantIgnored)
		}
	}
}

func TestDiagnosticsIgnoreErrorChecks(t *testing.T) {
	src := `package test

func Handle(err error) error {
	if err != nil {
		return err
	}
	return nil
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == "Handle" {
			fn = fd
			break
		}
	}
	if fn == nil {
		t.Fatal("Handle func not found")
	}

	resWithErr := gocognit.ScanComplexityWithOptions(fn, gocognit.ComplexityOptions{
		Diagnostics:       true,
		IgnoreErrorChecks: false,
	})
	if len(resWithErr.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic without flag, got %d", len(resWithErr.Diagnostics))
	}

	resIgnored := gocognit.ScanComplexityWithOptions(fn, gocognit.ComplexityOptions{
		Diagnostics:       true,
		IgnoreErrorChecks: true,
	})
	if len(resIgnored.Diagnostics) != 0 {
		t.Fatalf("expected 0 diagnostics with flag, got %d", len(resIgnored.Diagnostics))
	}
}

