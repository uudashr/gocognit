package gocognit

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"log"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	// logger = log.Default()
	logger = log.New(io.Discard, "", 0)
)

// Stat is statistic of the complexity.
type Stat struct {
	PkgName    string
	FuncName   string
	Complexity int
	BeginPos   token.Position
	EndPos     token.Position
}

func (s Stat) String() string {
	filePos := fmt.Sprintf("%s:%d,%d", s.BeginPos.Filename, s.BeginPos.Line, s.EndPos.Line)
	return fmt.Sprintf("%d %s %s %s", s.Complexity, s.PkgName, s.FuncName, filePos)
}

// ComplexityStats builds the complexity statistics.
func ComplexityStats(f *ast.File, fset *token.FileSet, stats []Stat) []Stat {
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			stats = append(stats, Stat{
				PkgName:    f.Name.Name,
				FuncName:   funcName(fn),
				Complexity: Complexity(fset, fn),
				BeginPos:   fset.Position(fn.Pos()),
				EndPos:     fset.Position(fn.End()),
			})
		}
	}
	return stats
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

// recvString returns a string representation of recv of the
// form "T", "*T", or "BADRECV" (if not a proper receiver type).
func recvString(recv ast.Expr) string {
	switch t := recv.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + recvString(t.X)
	}
	return "BADRECV"
}

func typeName(i interface{}) string {
	return reflect.TypeOf(i).String()
}

// Complexity calculates the cognitive complexity of a function.
func Complexity(fset *token.FileSet, fn *ast.FuncDecl) int {
	v := complexityVisitor{
		name: fn.Name,
	}

	logger.Printf("*** %s begin ***", v.name)

	ast.Walk(&v, fn)

	logger.Printf("*** %s end ***", v.name)
	logger.Print("")
	return v.complexity
}

type stmtType int

const (
	stmtTypeIf stmtType = iota
	stmtTypeSwitch
	stmtTypeCase
)

type complexityVisitor struct {
	fset            *token.FileSet
	name            *ast.Ident
	complexity      int
	nesting         int
	elseNodes       map[ast.Node]bool
	calculatedExprs map[ast.Expr]bool
}

func (v *complexityVisitor) printBody(n ast.Node) {
	if v.fset == nil {
		return
	}
	ast.Print(v.fset, n)
}

func (v *complexityVisitor) incNesting() {
	v.nesting++
}

func (v *complexityVisitor) decNesting() {
	v.nesting--
}

func (v *complexityVisitor) incComplexity() {
	logger.Printf("+1 incComplexity")
	v.complexity++
}

func (v *complexityVisitor) nestIncComplexity() {
	logger.Printf("+%d nestIncComplexity", v.nesting+1)
	v.complexity += (v.nesting + 1)
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
	if n == nil {
		return v
	}

	var fn func() ast.Visitor
	switch n := n.(type) {
	default:
		return v

	case *ast.IfStmt:
		fn = func() ast.Visitor { return v.visitIfStmt(n) }
	case *ast.SwitchStmt:
		fn = func() ast.Visitor { return v.visitSwitchStmt(n) }
	case *ast.TypeSwitchStmt:
		fn = func() ast.Visitor { return v.visitTypeSwitchStmt(n) }
	case *ast.SelectStmt:
		fn = func() ast.Visitor { return v.visitSelectStmt(n) }
	case *ast.ForStmt:
		fn = func() ast.Visitor { return v.visitForStmt(n) }
	case *ast.RangeStmt:
		fn = func() ast.Visitor { return v.visitRangeStmt(n) }
	case *ast.FuncLit:
		fn = func() ast.Visitor { return v.visitFuncLit(n) }
	case *ast.BranchStmt:
		fn = func() ast.Visitor { return v.visitBranchStmt(n) }
	case *ast.BinaryExpr:
		fn = func() ast.Visitor { return v.visitBinaryExpr(n) }
	case *ast.CallExpr:
		fn = func() ast.Visitor { return v.visitCallExpr(n) }
	case *ast.CaseClause:
		fn = func() ast.Visitor { return v.visitCaseClause(n) }
	}

	logger.Printf("%s begin", typeName(n))
	defer logger.Printf("%s end", typeName(n))
	res := fn()
	return res
}

func (v *complexityVisitor) visitIfStmt(n *ast.IfStmt) ast.Visitor {
	v.incIfComplexity(n)

	if t := n.Init; t != nil {
		logger.Print("if init begin")
		ast.Walk(v, t)
		logger.Print("if init end")
	}

	logger.Print("if cond begin")
	ast.Walk(v, n.Cond)
	logger.Print("if cond end")

	pure := !v.markedAsElseNode(n) // pure `if` statement, not an `else if`
	if pure {
		v.incNesting()
		ast.Walk(v, n.Body)
		v.decNesting()
	} else {
		ast.Walk(v, n.Body)
	}

	if _, ok := n.Else.(*ast.BlockStmt); ok {
		v.incComplexity()

		ast.Walk(v, n.Else)
	} else if _, ok := n.Else.(*ast.IfStmt); ok {
		v.markAsElseNode(n.Else)
		ast.Walk(v, n.Else)
	}

	return nil
}

func (v *complexityVisitor) visitSwitchStmt(n *ast.SwitchStmt) ast.Visitor {
	if n := n.Init; n != nil {
		logger.Print("switch init begin", typeName(n))
		ast.Walk(v, n)
		logger.Print("switch init end", typeName(n))
	}

	if n := n.Tag; n != nil {
		v.nestIncComplexity()
		logger.Print("switch tag begin", typeName(n))
		ast.Walk(v, n)
		logger.Print("switch tag end", typeName(n))
	}

	v.incNesting()

	logger.Print("switch body begin")
	ast.Walk(v, n.Body)
	logger.Print("switch body end")

	v.decNesting()
	return nil
}

func (v *complexityVisitor) visitTypeSwitchStmt(n *ast.TypeSwitchStmt) ast.Visitor {
	v.nestIncComplexity()

	if n := n.Init; n != nil {
		logger.Print("switch type init begin")
		ast.Walk(v, n)
		logger.Print("switch type init end")
	}

	if n := n.Assign; n != nil {
		logger.Print("switch type assign begin", typeName(n))
		ast.Walk(v, n)
		logger.Print("switch type assign end", typeName(n))
	}

	v.incNesting()
	logger.Print("switch type body begin")
	ast.Walk(v, n.Body)
	logger.Print("switch type body end")
	v.decNesting()
	return nil
}

func (v *complexityVisitor) visitSelectStmt(n *ast.SelectStmt) ast.Visitor {
	v.nestIncComplexity()

	v.incNesting()
	ast.Walk(v, n.Body)
	v.decNesting()
	return nil
}

func (v *complexityVisitor) visitForStmt(n *ast.ForStmt) ast.Visitor {
	v.nestIncComplexity()

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
	v.nestIncComplexity()

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
		v.incComplexity()
	}
	return v
}

func (v *complexityVisitor) visitCaseClause(n *ast.CaseClause) ast.Visitor {
	for _, expr := range n.List {
		n, _ := expr.(*ast.BinaryExpr)
		if n == nil {
			continue
		}

		// ast.Walk(v, expr)
		if v.isCalculated(n) {
			continue
		}

		v.incComplexity()
		v.visitBinaryExpr(n)
	}

	// for _, expr := range n.List {
	// 	if n, _ := expr.(*ast.BinaryExpr); n != nil {
	// 		v.incComplexity()
	// 		continue
	// 	}
	//
	// 	ast.Walk(v, expr)
	// }

	for _, n := range n.Body {
		ast.Walk(v, n)
	}
	return nil
}

func (v *complexityVisitor) visitBinaryExpr(n *ast.BinaryExpr) ast.Visitor {
	if v.isCalculated(n) {
		return v
	}
	if n.Op != token.LAND && n.Op != token.LOR {
		return v
	}

	ops := v.collectBinaryOps(n)
	for _, op := range ops {
		logger.Printf("op:%s", op.String())

		if op != token.LAND && op != token.LOR {
			continue
		}

		v.incComplexity()
	}

	return v
}

func (v *complexityVisitor) visitCallExpr(n *ast.CallExpr) ast.Visitor {
	if callIdent, ok := n.Fun.(*ast.Ident); ok {
		obj, name := callIdent.Obj, callIdent.Name
		if obj == v.name.Obj && name == v.name.Name {
			// called by same function directly (direct recursion)
			v.incComplexity()
		}
	}
	return v
}

func (v *complexityVisitor) collectBinaryOps(exp ast.Expr) []token.Token {
	v.markCalculated(exp)
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		return mergeBinaryOps(v.collectBinaryOps(exp.X), exp.Op, v.collectBinaryOps(exp.Y))
	case *ast.ParenExpr:
		// interest only on what inside paranthese
		return v.collectBinaryOps(exp.X)
	default:
		return []token.Token{}
	}
}

func (v *complexityVisitor) incIfComplexity(n *ast.IfStmt) {
	if v.markedAsElseNode(n) {
		v.incComplexity()
	} else {
		v.nestIncComplexity()
	}
}

func mergeBinaryOps(x []token.Token, op token.Token, y []token.Token) []token.Token {
	var out []token.Token
	if len(x) != 0 {
		out = append(out, x...)
	}
	out = append(out, op)
	if len(y) != 0 {
		out = append(out, y...)
	}
	return out
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
		fnDecl := n.(*ast.FuncDecl)

		fnName := funcName(fnDecl)
		fnComplexity := Complexity(nil, fnDecl)

		if fnComplexity > over {
			pass.Reportf(fnDecl.Pos(), "cognitive complexity %d of func %s is high (> %d)", fnComplexity, fnName, over)
		}
	})

	return nil, nil
}
