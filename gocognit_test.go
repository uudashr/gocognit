package gocognit

import (
	"go/ast"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func getAnalyzer() *analysis.Analyzer {
	const Doc = `Find complex function using cognitive complexity calculation.

The gocognit analysis reports functions or methods which the complexity is over
than the specified limit.`

	var over int // -over flag

	Analyzer := &analysis.Analyzer{
		Name:     "gocognit",
		Doc:      Doc,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

			nodeFilter := []ast.Node{
				(*ast.FuncDecl)(nil),
			}
			inspect.Preorder(nodeFilter, func(n ast.Node) {
				fnDecl := n.(*ast.FuncDecl)

				fnName := funcName(fnDecl)
				fnComplexity := Complexity(nil, fnDecl)

				if fnComplexity > over {
					pass.Reportf(fnDecl.Pos(), "cognitive complexity %d of func %s is high (> %d)", fnComplexity, fnName, over)
				}
			})

			return nil, nil
		},
	}

	Analyzer.Flags.IntVar(&over, "over", over, "show functions with complexity > N only")

	return Analyzer
}

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analyzer := getAnalyzer()
	analyzer.Flags.Set("over", "0")
	analysistest.Run(t, testdata, analyzer, "a")
}

func TestAnalyzerOver3(t *testing.T) {
	testdata := analysistest.TestData()
	analyzer := getAnalyzer()
	analyzer.Flags.Set("over", "3")
	analysistest.Run(t, testdata, analyzer, "b")
}
