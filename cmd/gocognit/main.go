// GoCognitive calculates the cognitive complexities of functions and
// methods in Go source code.
//
// Usage:
//
//	gocognit [<flag> ...] <Go file or directory> ...
//
// Flags:
//
//	-over N    show functions with complexity > N only and return exit code 1 if the output is non-empty
//	-top N     show the top N most complex functions only
//	-avg       show the average complexity over all functions, not depending on whether -over or -top are set
//	-test      indicates whether test files should be included
//	-json      encode the output as JSON
//	-d 	       enable diagnostic output
//	-f format  string the format to use (default "{{.Complexity}} {{.PkgName}} {{.FuncName}} {{.Pos}}")
//
// The (default) output fields for each line are:
//
//	<complexity> <package> <function> <file:row:column>
//
// The (default) output fields for each line are:
//
//	{{.Complexity}} {{.PkgName}} {{.FuncName}} {{.Pos}}
//
// or equal to <complexity> <package> <function> <file:row:column>
//
// The struct being passed to the template is:
//
//	type Stat struct {
//	  PkgName    string
//	  FuncName   string
//	  Complexity int
//	  Diagnostics []Diagnostic
//	  Pos        token.Position
//	}
//
//	type Diagnostic struct {
//	  Inc     string
//	  Nesting int
//	  Text    string
//	  Pos     DiagnosticPosition
//	}
//
//	type DiagnosticPosition struct {
//	  Offset int
//	  Line   int
//	  Column int
//	}
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/uudashr/gocognit"
	"golang.org/x/tools/go/packages"
)

const usageDoc = `Calculate cognitive complexities of Go functions.

Usage:

  gocognit [<flag> ...] <Go file or directory> ...

Flags:

  -over N           show functions with complexity > N only
                    and return exit code 1 if the output is non-empty
  -top N            show the top N most complex functions only
  -avg              show the average complexity over all functions,
                    not depending on whether -over or -top are set
  -test             indicates whether test files should be included
  -json             encode the output as JSON
  -d                enable diagnostic output
  -f format         string the format to use
                    (default "{{.Complexity}} {{.PkgName}} {{.FuncName}} {{.Pos}}")
  -ignore expr      ignore files matching the given regexp
  -exact-recursion  work out what each call points to, the way the compiler
                    does, so recursion is scored exactly. Slower, and the code
                    must build; without it, mutually recursive methods may be
                    under-counted

The (default) output fields for each line are:

  <complexity> <package> <function> <file:row:column>

The (default) output fields for each line are:

  {{.Complexity}} {{.PkgName}} {{.FuncName}} {{.Pos}}

or equal to <complexity> <package> <function> <file:row:column>

The struct being passed to the template is:

  type Stat struct {
    PkgName     string
    FuncName    string
    Complexity  int
    Pos         token.Position
    Diagnostics []Diagnostics
  }

  type Diagnostic struct {
    Inc     string
    Nesting int
    Text    string
    Pos     DiagnosticPosition
  }	

  type DiagnosticPosition struct {
    Offset int
    Line   int
    Column int
  }
`

const (
	defaultOverFlagVal = 0
	defaultTopFlagVal  = -1
)

const defaultFormat = "{{.Complexity}} {{.PkgName}} {{.FuncName}} {{.Pos}}"

func usage() {
	_, _ = fmt.Fprint(os.Stderr, usageDoc)
	os.Exit(2)
}

func main() {
	var (
		over              int
		top               int
		avg               bool
		includeTests      bool
		format            string
		jsonEncode        bool
		enableDiagnostics bool
		ignoreExpr        string
		exactRecursion    bool
	)

	flag.IntVar(&over, "over", defaultOverFlagVal, "show functions with complexity > N only")
	flag.IntVar(&top, "top", defaultTopFlagVal, "show the top N most complex functions only")
	flag.BoolVar(&avg, "avg", false, "show the average complexity")
	flag.BoolVar(&includeTests, "test", true, "indicates whether test files should be included")
	flag.StringVar(&format, "f", defaultFormat, "the format to use")
	flag.BoolVar(&jsonEncode, "json", false, "encode the output as JSON")
	flag.BoolVar(&enableDiagnostics, "d", false, "enable diagnostic output")
	flag.StringVar(&ignoreExpr, "ignore", "", "ignore files matching the given regexp")
	flag.BoolVar(&exactRecursion, "exact-recursion", false, "work out what each call points to, like the compiler, so recursion is scored exactly (slower; the code must build)")

	log.SetFlags(0)
	log.SetPrefix("gocognit: ")

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		usage()
	}

	tmpl, err := template.New("gocognit").Parse(format)
	if err != nil {
		log.Fatal(err)
	}

	stats, err := analyze(args, includeTests, enableDiagnostics, exactRecursion)
	if err != nil {
		log.Fatal(err)
	}

	sort.Sort(byComplexity(stats))

	ignoreRegexp, err := prepareRegexp(ignoreExpr)
	if err != nil {
		log.Fatal(err)
	}

	filteredStats := filterStats(stats, ignoreRegexp, top, over)

	var written int
	if jsonEncode {
		written, err = writeJSONStats(os.Stdout, filteredStats)
	} else {
		written, err = writeTextStats(os.Stdout, filteredStats, tmpl)
	}

	if err != nil {
		log.Fatal(err)
	}

	if avg {
		showAverage(stats)
	}

	if over > 0 && written > 0 {
		os.Exit(1)
	}
}

func analyzePath(path string, includeTests bool, includeDiagnostic bool, exactRecursion bool) ([]gocognit.Stat, error) {
	if exactRecursion {
		stats, err := analyzeExactRecursionPath(path, includeTests, includeDiagnostic)
		if err == nil {
			return stats, nil
		}

		log.Printf("warning: %s: type-aware analysis failed, falling back to syntax: %v", path, err)

		// The user already asked for type-aware analysis; do not suggest it again.
		return analyzeSyntacticPath(path, includeTests, includeDiagnostic, false)
	}

	return analyzeSyntacticPath(path, includeTests, includeDiagnostic, true)
}

func analyzeSyntacticPath(path string, includeTests bool, includeDiagnostic bool, suggestExactRecursion bool) ([]gocognit.Stat, error) {
	if isDir(path) {
		return analyzeDir(path, includeTests, nil, includeDiagnostic, suggestExactRecursion)
	}

	return analyzeFile(path, nil, includeDiagnostic, suggestExactRecursion)
}

func analyze(paths []string, includeTests bool, includeDiagnostic bool, exactRecursion bool) (stats []gocognit.Stat, err error) {
	var out []gocognit.Stat

	for _, path := range paths {
		stats, err := analyzePath(path, includeTests, includeDiagnostic, exactRecursion)
		if err != nil {
			return nil, err
		}

		out = append(out, stats...)
	}

	return out, nil
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)

	return err == nil && fi.IsDir()
}

// exactRecursionLoadMode is what go/packages needs to hand out a usable types.Info.
// NeedImports and NeedDeps matter: without them, loading a package that imports
// anything fails with `package "fmt" without types was imported`.
const exactRecursionLoadMode = packages.NeedName | packages.NeedFiles | packages.NeedImports |
	packages.NeedDeps | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo

// analyzeExactRecursionPath scores path using type information, so calls are resolved
// exactly (including mutual method recursion). It returns an error when path
// cannot be loaded with types, letting the caller fall back to syntax.
func analyzeExactRecursionPath(path string, includeTests bool, includeDiagnostic bool) ([]gocognit.Stat, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode:  exactRecursionLoadMode,
		Tests: includeTests,
	}

	isFile := !isDir(path)
	if isFile {
		cfg.Dir = filepath.Dir(absPath)
	} else {
		cfg.Dir = absPath
	}

	pattern := "./..."
	if isFile {
		pattern = "file=" + absPath
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, nil
	}

	var (
		stats     []gocognit.Stat
		usable    bool
		hadErrors bool
	)

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			hadErrors = true
		}

		// Skip packages without full type information, dependency packages,
		// and the generated test-main package (whose only file lives in the
		// build cache, outside the requested path).
		if pkg.TypesInfo == nil || pkg.Types == nil || !packageWithinTarget(pkg, absPath, isFile) {
			continue
		}

		usable = true
		stats = gocognit.ComplexityStatsForFilesWithDiagnostic(pkg.Syntax, pkg.Fset, pkg.TypesInfo, stats, includeDiagnostic)
	}

	if !usable {
		return nil, fmt.Errorf("no type information available for %s", path)
	}

	if hadErrors {
		log.Printf("warning: %s: some packages have type errors, results may be incomplete", path)
	}

	return filterStatFiles(dedupeStats(stats), absPath, isFile), nil
}

// packageWithinTarget reports whether pkg owns a file inside the requested
// path. This excludes dependencies and the generated test-main package.
func packageWithinTarget(pkg *packages.Package, target string, isFile bool) bool {
	for _, filename := range pkg.GoFiles {
		if statFileWithinTarget(filename, target, isFile) {
			return true
		}
	}

	return false
}

func statFileWithinTarget(filename, target string, isFile bool) bool {
	filename = filepath.Clean(filename)
	target = filepath.Clean(target)

	if isFile {
		return filename == target
	}

	return filename == target || strings.HasPrefix(filename, target+string(filepath.Separator))
}

// filterStatFiles keeps only the statistics for the requested path, so that a
// single-file argument reports just that file even though the whole package was
// needed to resolve its calls.
func filterStatFiles(stats []gocognit.Stat, target string, isFile bool) []gocognit.Stat {
	out := stats[:0]

	for _, stat := range stats {
		if statFileWithinTarget(stat.Pos.Filename, target, isFile) {
			out = append(out, stat)
		}
	}

	return out
}

// dedupeStats removes duplicates produced by the test variants go/packages
// returns (the base package, the internal-test variant, and the external-test
// variant all contain the same non-test files).
func dedupeStats(stats []gocognit.Stat) []gocognit.Stat {
	type key struct {
		filename string
		line     int
		column   int
		funcName string
	}

	seen := make(map[key]bool, len(stats))
	out := stats[:0]

	for _, stat := range stats {
		k := key{stat.Pos.Filename, stat.Pos.Line, stat.Pos.Column, stat.FuncName}
		if seen[k] {
			continue
		}

		seen[k] = true
		out = append(out, stat)
	}

	return out
}

func analyzeFile(fname string, stats []gocognit.Stat, includeDiagnostic bool, suggestExactRecursion bool) ([]gocognit.Stat, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	if suggestExactRecursion {
		suggestExactRecursionFor([]*ast.File{f})
	}

	return gocognit.ComplexityStatsWithDiagnostic(f, fset, stats, includeDiagnostic), nil
}

func analyzeDir(dirname string, includeTests bool, stats []gocognit.Stat, includeDiagnostic bool, suggestExactRecursion bool) ([]gocognit.Stat, error) {
	byDir := make(map[string][]string)

	err := filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}

		dir := filepath.Dir(path)
		byDir[dir] = append(byDir[dir], path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		stats, err = analyzeDirFiles(byDir[dir], stats, includeDiagnostic, suggestExactRecursion)
		if err != nil {
			return nil, err
		}
	}

	return stats, nil
}

// analyzeDirFiles parses and analyzes the Go files that live in one directory.
// Files are grouped by package so that recursion can be detected across the
// files of a package instead of one file at a time.
func analyzeDirFiles(filenames []string, stats []gocognit.Stat, includeDiagnostic bool, suggestExactRecursion bool) ([]gocognit.Stat, error) {
	sort.Strings(filenames)

	fset := token.NewFileSet()

	byPkg := make(map[string][]*ast.File)
	var order []string

	for _, filename := range filenames {
		f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		name := f.Name.Name
		if _, ok := byPkg[name]; !ok {
			order = append(order, name)
		}

		byPkg[name] = append(byPkg[name], f)
	}

	for _, name := range order {
		if suggestExactRecursion {
			suggestExactRecursionFor(byPkg[name])
		}

		stats = gocognit.ComplexityStatsForFilesWithDiagnostic(byPkg[name], fset, nil, stats, includeDiagnostic)
	}

	return stats, nil
}

// exactRecursionHintShown guards the one-time note suggesting -exact-recursion. The CLI runs
// single-threaded, so a plain bool is enough.
var exactRecursionHintShown bool

// suggestExactRecursion prints, at most once per run, a note when the type-less
// analysis may have under-counted method recursion. The note goes to stderr so
// that JSON output on stdout stays clean.
func suggestExactRecursionFor(files []*ast.File) {
	if exactRecursionHintShown || !gocognit.PossibleMethodRecursion(files) {
		return
	}

	exactRecursionHintShown = true
	log.Print("note: some calls could not be resolved from the source alone; recursion may be under-counted. Re-run with -exact-recursion for exact results")
}

func writeTextStats(w io.Writer, stats []gocognit.Stat, tmpl *template.Template) (int, error) {
	for i, stat := range stats {
		if err := tmpl.Execute(w, stat); err != nil {
			return i, err
		}

		fmt.Fprintln(w)
	}

	return len(stats), nil
}

func writeJSONStats(w io.Writer, stats []gocognit.Stat) (int, error) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	if err := enc.Encode(stats); err != nil {
		return 0, err
	}

	return len(stats), nil
}

func prepareRegexp(expr string) (*regexp.Regexp, error) {
	if expr == "" {
		return nil, nil
	}

	return regexp.Compile(expr)
}

func filterStats(sortedStats []gocognit.Stat, ignoreRegexp *regexp.Regexp, top, over int) []gocognit.Stat {
	var filtered []gocognit.Stat

	i := 0
	for _, stat := range sortedStats {
		if i == top {
			break
		}

		if stat.Complexity <= over {
			break
		}

		if ignoreRegexp != nil && ignoreRegexp.MatchString(stat.Pos.Filename) {
			continue
		}

		filtered = append(filtered, stat)
		i++
	}

	return filtered
}

func showAverage(stats []gocognit.Stat) {
	fmt.Printf("Average: %.3g\n", average(stats))
}

func average(stats []gocognit.Stat) float64 {
	total := 0
	for _, s := range stats {
		total += s.Complexity
	}

	return float64(total) / float64(len(stats))
}

type byComplexity []gocognit.Stat

func (s byComplexity) Len() int      { return len(s) }
func (s byComplexity) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byComplexity) Less(i, j int) bool {
	return s[i].Complexity >= s[j].Complexity
}
