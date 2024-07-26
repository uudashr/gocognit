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
//	-json      encode the output as JSON
//	-f format  string the format to use (default "{{.PkgName}}.{{.FuncName}}:{{.Complexity}}:{{.Pos}}")
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
//	  Pos        token.Position
//	}
package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
)

const usageDoc = `Calculate cognitive complexities of Go functions.

Usage:

  gocognit [<flag> ...] <Go file or directory> ...

Flags:

  -over N    show functions with complexity > N only
             and return exit code 1 if the output is non-empty
  -top N     show the top N most complex functions only
  -avg       show the average complexity over all functions,
             not depending on whether -over or -top are set
  -test      indicates whether test files should be included
  -json      encode the output as JSON
  -f format  string the format to use 
             (default "{{.PkgName}}.{{.FuncName}}:{{.Complexity}}:{{.Pos}}")

The (default) output fields for each line are:

  <complexity> <package> <function> <file:row:column>

The (default) output fields for each line are:

  {{.Complexity}} {{.PkgName}} {{.FuncName}} {{.Pos}}

or equal to <complexity> <package> <function> <file:row:column>

The struct being passed to the template is:

  type Stat struct {
    PkgName    string
    FuncName   string
    Complexity int
    Pos        token.Position
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
		over         int
		top          int
		avg          bool
		includeTests bool
		format       string
		jsonEncode   bool
		ignoreExpr   string
	)
	flag.IntVar(&over, "over", defaultOverFlagVal, "show functions with complexity > N only")
	flag.IntVar(&top, "top", defaultTopFlagVal, "show the top N most complex functions only")
	flag.BoolVar(&avg, "avg", false, "show the average complexity")
	flag.BoolVar(&includeTests, "test", true, "indicates whether test files should be included")
	flag.StringVar(&format, "f", defaultFormat, "the format to use")
	flag.BoolVar(&jsonEncode, "json", false, "encode the output as JSON")
	flag.StringVar(&ignoreExpr, "ignore", "", "ignore files matching the given regexp")

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

	stats, err := analyze(args, includeTests)
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

func analyzePath(path string, includeTests bool) ([]gocognit.Stat, error) {
	if isDir(path) {
		return analyzeDir(path, includeTests, nil)
	}

	return analyzeFile(path, nil)
}

func analyze(paths []string, includeTests bool) (stats []gocognit.Stat, err error) {
	var out []gocognit.Stat
	for _, path := range paths {
		stats, err := analyzePath(path, includeTests)
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

func analyzeFile(fname string, stats []gocognit.Stat) ([]gocognit.Stat, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	return gocognit.ComplexityStats(f, fset, stats), nil
}

func analyzeDir(dirname string, includeTests bool, stats []gocognit.Stat) ([]gocognit.Stat, error) {
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

		stats, err = analyzeFile(path, stats)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
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
