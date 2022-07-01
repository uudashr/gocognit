//go:build go1.18
// +build go1.18

package gocognit_test

import (
	"testing"

	"github.com/uudashr/gocognit"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer_Generics(t *testing.T) {
	testdata := analysistest.TestData()
	gocognit.Analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, gocognit.Analyzer, "c")
}
