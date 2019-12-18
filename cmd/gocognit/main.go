// GoCognitive calculates the cognitive complexities of functions and
// methods in Go source code.
//
// Usage:
//      gocognitive [<flag> ...] <Go file or directory> ...
//
// Flags:
//      -over N   show functions with complexity > N only and
//                return exit code 1 if the output is non-empty
//      -top N    show the top N most complex functions only
//      -avg      show the average complexity
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
The output fields for each line are:
<complexity> <package> <function> <file:row:column>
`

func usage() {
	fmt.Fprint(os.Stderr, usageDoc)
	os.Exit(2)
}

var (
	over = flag.Int("over", 0, "show functions with complexity > N only")
	top  = flag.Int("top", -1, "show the top N most complex functions only")
	avg  = flag.Bool("avg", false, "show the average complexity")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("cognitive: ")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
	}

	stats := analyze(args)
	sort.Sort(byComplexity(stats))
	written := writeStats(os.Stdout, stats)

	if *avg {
		showAverage(stats)
	}

	if *over > 0 && written > 0 {
		os.Exit(1)
	}
}

func analyze(paths []string) []gocognit.Stat {
	var stats []gocognit.Stat
	for _, path := range paths {
		if isDir(path) {
			stats = analyzeDir(path, stats)
		} else {
			stats = analyzeFile(path, stats)
		}
	}

	return stats
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func analyzeFile(fname string, stats []gocognit.Stat) []gocognit.Stat {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	return gocognit.ComplexityStats(f, fset, stats)
}

func analyzeDir(dirname string, stats []gocognit.Stat) []gocognit.Stat {
	err := filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
			stats = analyzeFile(path, stats)
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	return stats
}

func writeStats(w io.Writer, sortedStats []gocognit.Stat) int {
	for i, stat := range sortedStats {
		if i == *top {
			return i
		}
		if stat.Complexity <= *over {
			return i
		}
		fmt.Fprintln(w, stat)
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
