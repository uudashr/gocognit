package gocognit

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	Analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, Analyzer, "a")
}

func TestAnalyzerOver3(t *testing.T) {
	testdata := analysistest.TestData()
	Analyzer.Flags.Set("over", "3")
	analysistest.Run(t, testdata, Analyzer, "b")
}
