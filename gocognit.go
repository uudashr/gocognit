package gocognit

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Stat is statistic of the complexity.
type Stat struct {
	PkgName    string
	FuncName   string
	Complexity int
	Traces     []Trace `json:",omitempty"`
	Pos        token.Position
}

type Trace struct {
	Inc     int
	Nesting int `json:",omitempty"`
	Text    string
	Pos     token.Position
}

func (t Trace) String() string {
	if t.Nesting == 0 {
		return fmt.Sprintf("+%d", t.Inc)
	}

	return fmt.Sprintf("+%d (nesting=%d)", t.Inc, t.Nesting)
}

func (s Stat) String() string {
	return fmt.Sprintf("%d %s %s %s", s.Complexity, s.PkgName, s.FuncName, s.Pos)
}

// ComplexityStats builds the complexity statistics.
func ComplexityStats(f *ast.File, fset *token.FileSet, stats []Stat, includeTrace bool) []Stat {
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			d := parseDirective(fn.Doc)
			if d.Ignore {
				continue
			}

			res := ScanComplexity(fn, includeTrace)

			stats = append(stats, Stat{
				PkgName:    f.Name.Name,
				FuncName:   funcName(fn),
				Complexity: res.Complexity,
				Traces:     traces(fset, res.Traces),
				Pos:        fset.Position(fn.Pos()),
			})
		}
	}

	return stats
}

func traces(fset *token.FileSet, traces []trace) []Trace {
	tags := make([]Trace, 0, len(traces))
	for _, t := range traces {
		tags = append(tags, Trace{
			Inc:     t.Inc,
			Nesting: t.Nesting,
			Text:    t.Text,
			Pos:     fset.Position(t.Pos),
		})
	}

	return tags
}

type directive struct {
	Ignore bool
}

func parseDirective(doc *ast.CommentGroup) directive {
	if doc == nil {
		return directive{}
	}

	for _, c := range doc.List {
		if c.Text == "//gocognit:ignore" {
			return directive{Ignore: true}
		}
	}

	return directive{}
}

// funcName returns the name representation of a function or method:
// "(Type).Name" for methods or simply "Name" for functions.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil {
		if fn.Recv.NumFields() > 0 {
			typ := fn.Recv.List[0].Type

			return fmt.Sprintf("(%s).%s", recvString(typ), fn.Name)
		}
	}

	return fn.Name.Name
}

// Complexity calculates the cognitive complexity of a function.
func Complexity(fn *ast.FuncDecl) int {
	res := ScanComplexity(fn, false)

	return res.Complexity
}

// ScanComplexity scans the function declaration.
func ScanComplexity(fn *ast.FuncDecl, trace bool) ScanResult {
	v := complexityVisitor{
		name:  fn.Name,
		trace: trace,
	}

	ast.Walk(&v, fn)

	return ScanResult{
		Traces:     v.traces,
		Complexity: v.complexity,
	}
}

type ScanResult struct {
	Traces     []trace
	Complexity int
}

type trace struct {
	Inc     int
	Nesting int
	Text    string
	Pos     token.Pos
}

type complexityVisitor struct {
	name            *ast.Ident
	complexity      int
	nesting         int
	elseNodes       map[ast.Node]bool
	calculatedExprs map[ast.Expr]bool

	fset *token.FileSet

	trace  bool
	traces []trace
}

func (v *complexityVisitor) incNesting() {
	v.nesting++
}

func (v *complexityVisitor) decNesting() {
	v.nesting--
}

func (v *complexityVisitor) incComplexity(text string, pos token.Pos) {
	v.complexity++

	if !v.trace {
		return
	}

	v.traces = append(v.traces, trace{
		Inc:  1,
		Text: text,
		Pos:  pos,
	})
}

func (v *complexityVisitor) nestIncComplexity(text string, pos token.Pos) {
	v.complexity += (v.nesting + 1)

	if !v.trace {
		return
	}

	v.traces = append(v.traces, trace{
		Inc:     v.nesting + 1,
		Nesting: v.nesting,
		Text:    text,
		Pos:     pos,
	})
}

func (v *complexityVisitor) markAsElseNode(n ast.Node) {
	if v.elseNodes == nil {
		v.elseNodes = make(map[ast.Node]bool)
	}

	v.elseNodes[n] = true
}

func (v *complexityVisitor) markedAsElseNode(n ast.Node) bool {
	if v.elseNodes == nil {
		return false
	}

	return v.elseNodes[n]
}

func (v *complexityVisitor) markCalculated(e ast.Expr) {
	if v.calculatedExprs == nil {
		v.calculatedExprs = make(map[ast.Expr]bool)
	}

	v.calculatedExprs[e] = true
}

func (v *complexityVisitor) isCalculated(e ast.Expr) bool {
	if v.calculatedExprs == nil {
		return false
	}

	return v.calculatedExprs[e]
}

// Visit implements the ast.Visitor interface.
func (v *complexityVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.IfStmt:
		return v.visitIfStmt(n)
	case *ast.SwitchStmt:
		return v.visitSwitchStmt(n)
	case *ast.TypeSwitchStmt:
		return v.visitTypeSwitchStmt(n)
	case *ast.SelectStmt:
		return v.visitSelectStmt(n)
	case *ast.ForStmt:
		return v.visitForStmt(n)
	case *ast.RangeStmt:
		return v.visitRangeStmt(n)
	case *ast.FuncLit:
		return v.visitFuncLit(n)
	case *ast.BranchStmt:
		return v.visitBranchStmt(n)
	case *ast.BinaryExpr:
		return v.visitBinaryExpr(n)
	case *ast.CallExpr:
		return v.visitCallExpr(n)
	}

	return v
}

func (v *complexityVisitor) visitIfStmt(n *ast.IfStmt) ast.Visitor {
	v.incIfComplexity(n, "if", n.Pos())

	if n := n.Init; n != nil {
		ast.Walk(v, n)
	}

	ast.Walk(v, n.Cond)

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()

	if _, ok := n.Else.(*ast.BlockStmt); ok {
		v.incComplexity("else", n.Else.Pos())

		ast.Walk(v, n.Else)
	} else if _, ok := n.Else.(*ast.IfStmt); ok {
		v.markAsElseNode(n.Else)
		ast.Walk(v, n.Else)
	}

	return nil
}

func (v *complexityVisitor) visitSwitchStmt(n *ast.SwitchStmt) ast.Visitor {
	v.nestIncComplexity("switch", n.Pos())

	if n := n.Init; n != nil {
		ast.Walk(v, n)
	}

	if n := n.Tag; n != nil {
		ast.Walk(v, n)
	}

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()

	return nil
}

func (v *complexityVisitor) visitTypeSwitchStmt(n *ast.TypeSwitchStmt) ast.Visitor {
	v.nestIncComplexity("switch", n.Pos())

	if n := n.Init; n != nil {
		ast.Walk(v, n)
	}

	if n := n.Assign; n != nil {
		ast.Walk(v, n)
	}

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()

	return nil
}

func (v *complexityVisitor) visitSelectStmt(n *ast.SelectStmt) ast.Visitor {
	v.nestIncComplexity("select", n.Pos())

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()

	return nil
}

func (v *complexityVisitor) visitForStmt(n *ast.ForStmt) ast.Visitor {
	v.nestIncComplexity("for", n.Pos())

	if n := n.Init; n != nil {
		ast.Walk(v, n)
	}

	if n := n.Cond; n != nil {
		ast.Walk(v, n)
	}

	if n := n.Post; n != nil {
		ast.Walk(v, n)
	}

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()

	return nil
}

func (v *complexityVisitor) visitRangeStmt(n *ast.RangeStmt) ast.Visitor {
	v.nestIncComplexity("for", n.Pos())

	if n := n.Key; n != nil {
		ast.Walk(v, n)
	}

	if n := n.Value; n != nil {
		ast.Walk(v, n)
	}

	ast.Walk(v, n.X)

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()

	return nil
}

func (v *complexityVisitor) visitFuncLit(n *ast.FuncLit) ast.Visitor {
	ast.Walk(v, n.Type)

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()
	return nil
}

func (v *complexityVisitor) visitBranchStmt(n *ast.BranchStmt) ast.Visitor {
	if n.Label != nil {
		v.incComplexity(n.Tok.String(), n.Pos())
	}
	return v
}

func (v *complexityVisitor) visitBinaryExpr(n *ast.BinaryExpr) ast.Visitor {
	if isBinaryLogicalOp(n.Op) && !v.isCalculated(n) {
		ops := v.collectBinaryOps(n)

		var lastOp token.Token
		for _, op := range ops {
			if lastOp != op {
				v.incComplexity(op.String(), n.OpPos)
				lastOp = op
			}
		}
	}

	return v
}

func (v *complexityVisitor) visitCallExpr(n *ast.CallExpr) ast.Visitor {
	if callIdent, ok := n.Fun.(*ast.Ident); ok {
		obj, name := callIdent.Obj, callIdent.Name
		if obj == v.name.Obj && name == v.name.Name {
			// called by same function directly (direct recursion)
			v.incComplexity(name, n.Pos())
		}
	}

	return v
}

func (v *complexityVisitor) collectBinaryOps(exp ast.Expr) []token.Token {
	v.markCalculated(exp)

	if exp, ok := exp.(*ast.BinaryExpr); ok {
		return mergeBinaryOps(v.collectBinaryOps(exp.X), exp.Op, v.collectBinaryOps(exp.Y))
	}
	return nil
}

func (v *complexityVisitor) incIfComplexity(n *ast.IfStmt, text string, pos token.Pos) {
	if v.markedAsElseNode(n) {
		v.incComplexity(text, pos)
	} else {
		v.nestIncComplexity(text, pos)
	}
}

func mergeBinaryOps(x []token.Token, op token.Token, y []token.Token) []token.Token {
	var out []token.Token
	out = append(out, x...)

	if isBinaryLogicalOp(op) {
		out = append(out, op)
	}

	out = append(out, y...)
	return out
}

func isBinaryLogicalOp(op token.Token) bool {
	return op == token.LAND || op == token.LOR
}

const Doc = `Find complex function using cognitive complexity calculation.

The gocognit analysis reports functions or methods which the complexity is over 
than the specified limit.`

// Analyzer reports a diagnostic for every function or method which is
// too complex specified by its -over flag.
var Analyzer = &analysis.Analyzer{
	Name:     "gocognit",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var (
	over int // -over flag
)

func init() {
	Analyzer.Flags.IntVar(&over, "over", over, "show functions with complexity > N only")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		d := parseDirective(funcDecl.Doc)
		if d.Ignore {
			return
		}

		fnName := funcName(funcDecl)

		fnComplexity := Complexity(funcDecl)

		if fnComplexity > over {
			pass.Reportf(funcDecl.Pos(), "cognitive complexity %d of func %s is high (> %d)", fnComplexity, fnName, over)
		}
	})

	return nil, nil
}
