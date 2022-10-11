// GoCognitive calculates the cognitive complexities of functions and
// methods in Go source code.
//
// Usage:
//
//	gocognitive [<flag> ...] <Go file or directory> ...
//
// Flags:
//
//	-over N   show functions with complexity > N only and
//	          return exit code 1 if the output is non-empty
//	-top N    show the top N most complex functions only
//	-avg      show the average complexity
//
// The output fields for each line are:
// <complexity> <package> <function> <file:row:column>
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/uudashr/gocognit"
)

const usageDoc = `Calculate cognitive complexities of Go functions.
Usage:
        gocognit [flags] <Go file or directory> ...
Flags:
        -over N   show functions with complexity > N only and
                  return exit code 1 if the set is non-empty
        -top N    show the top N most complex functions only
        -avg      show the average complexity over all functions,
                  not depending on whether -over or -top are set
        -format string
                  which format to use, supported formats: [text json json-pretty] (default "text")
The output fields for each line are:
<complexity> <package> <function> <file:row:column>
`

func usage() {
	_, _ = fmt.Fprint(os.Stderr, usageDoc)
	os.Exit(2)
}

const (
	defaultValueIndicator = -1
	textFormat            = "text"
	jsonFormat            = "json"
	jsonPrettyFormat      = "json-pretty"
)

var (
	supportedFormats = []string{
		textFormat, jsonFormat, jsonPrettyFormat,
	}

	over   = flag.Int("over", defaultValueIndicator, "show functions with complexity > N only")
	top    = flag.Int("top", defaultValueIndicator, "show the top N most complex functions only")
	avg    = flag.Bool("avg", false, "show the average complexity")
	format = flag.String("format", "text", fmt.Sprintf("which format to use, supported formats: %v", supportedFormats))
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("gocognit: ")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
	}

	stats, err := analyze(args)
	if err != nil {
		log.Fatal(err)
	}

	sort.Sort(byComplexity(stats))
	written := writeStats(os.Stdout, stats)

	if *avg {
		showAverage(stats)
	}

	if *over > 0 && written > 0 {
		os.Exit(1)
	}
}

func analyzePath(path string) ([]gocognit.Stat, error) {
	if isDir(path) {
		return analyzeDir(path, nil)
	}

	return analyzeFile(path, nil)
}

func analyze(paths []string) ([]gocognit.Stat, error) {
	var (
		stats []gocognit.Stat
		err   error
	)
	for _, path := range paths {
		stats, err = analyzePath(path)
		if err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func analyzeFile(fname string, stats []gocognit.Stat) ([]gocognit.Stat, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, 0)
	if err != nil {
		return nil, err
	}

	return gocognit.ComplexityStats(f, fset, stats), nil
}

func analyzeDir(dirname string, stats []gocognit.Stat) ([]gocognit.Stat, error) {
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

func writeStats(w io.Writer, sortedStats []gocognit.Stat) int {
	filter := gocognit.Filter{}
	if *top != defaultValueIndicator {
		// top filter
		filter.AddFilter(gocognit.NewTopFilter(*top))
	}

	if *over != defaultValueIndicator {
		// over filter
		filter.AddFilter(gocognit.NewComplexityFilter(*over))
	}

	var formatter gocognit.Formatter

	switch *format {
	case textFormat:
		formatter = gocognit.NewTextFormatter(w)
		break
	case jsonFormat:
		formatter = gocognit.NewJsonFormatter(w, false)
		break
	case jsonPrettyFormat:
		formatter = gocognit.NewJsonFormatter(w, true)
		break
	default:
		fmt.Printf("Format '%s' is not valid, use a supported format %v", *format, supportedFormats)
		os.Exit(1)
	}

	filtered := filter.Apply(sortedStats)

	err := formatter.Write(filtered)
	if err != nil {
		panic(err)
	}

	return len(sortedStats)
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
