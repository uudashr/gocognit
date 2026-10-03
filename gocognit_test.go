package gocognit_test

import (
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
