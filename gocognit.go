package gocognit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"reflect"
)

var (
	_logDiscard = log.New(ioDiscard{}, "", 0)
	_debug      bool
)

func SetDebug(enable bool) { _debug = enable }

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
	l := log.New(ioDiscard{}, "", 0)
	if _debug {
		l = log.New(os.Stdout, fmt.Sprintf("debug %s ", funcName(fn)),
			log.Lshortfile|log.LstdFlags)
	}
	v := complexityVisitor{
		log:  l,
		fset: fset,
		name: fn.Name,
	}

	v.log.Printf("***** %s begin *****", v.name)

	ast.Walk(&v, fn)

	v.log.Printf("***** %s end *****", v.name)
	v.log.Print("")
	return v.complexity
}

type complexityVisitor struct {
	log             *log.Logger
	fset            *token.FileSet
	name            *ast.Ident
	complexity      int
	nesting         int
	elseNodes       map[ast.Node]bool
	calculatedExprs map[ast.Expr]bool
	level           int
}

func (v *complexityVisitor) printBody(n ast.Node) {
	if v.fset == nil {
		return
	}
	buffer := bytes.NewBuffer(nil)
	ast.Fprint(buffer, v.fset, n, nil)
	v.log.Printf("print content: \n%s", buffer.String())
}

func (v *complexityVisitor) incLevel() string {
	v.level++
	s := fmt.Sprintf("L%d", v.level)
	return s
}

func (v *complexityVisitor) decLevel() string {
	s := fmt.Sprintf("L%d", v.level)
	v.level--
	return s
}

func (v *complexityVisitor) incNesting() {
	v.nesting++
	v.log.Printf("nesting +1. after: %d", v.nesting)
}

func (v *complexityVisitor) decNesting() {
	v.log.Printf("nesting -1. before: %d", v.nesting)
	v.nesting--
}

func (v *complexityVisitor) incComplexity() {
	v.log.Printf("*** incComplexity +1")
	v.complexity++
}

func (v *complexityVisitor) nestIncComplexity() {
	v.log.Printf("*** nestIncComplexity +%d", v.nesting+1)
	v.complexity += (v.nesting + 1)
}

func (v *complexityVisitor) nestIncComplexityOnly() {
	v.log.Printf("*** nestIncComplexityOnly +%d", v.nesting)
	v.complexity += v.nesting
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
	}

	v.log.Printf("%s begin %s", typeName(n), v.incLevel())
	defer func() {
		v.log.Printf("%s end %s", typeName(n), v.decLevel())
	}()
	res := fn()
	return res
}

func (v *complexityVisitor) visitIfStmt(n *ast.IfStmt) ast.Visitor {
	v.incIfComplexity(n)

	if t := n.Init; t != nil {
		v.log.Print("if init begin ", v.incLevel())
		ast.Walk(v, t)
		v.log.Print("if init end ", v.decLevel())
	}

	v.log.Print("if cond begin ", v.incLevel())
	ast.Walk(v, n.Cond)
	v.log.Print("if cond end ", v.decLevel())

	v.incNesting()
	v.log.Print("if body begin ", v.incLevel())
	ast.Walk(v, n.Body)
	v.log.Print("if body end ", v.decLevel())
	v.decNesting()

	switch t := n.Else.(type) {
	case *ast.BlockStmt:
		v.incComplexity()

		v.log.Print("if else block begin ", v.incLevel())
		// v.printBody(t)
		ast.Walk(v, t)
		v.log.Print("if else block end ", v.decLevel())

	case *ast.IfStmt:
		v.markAsElseNode(t)
		v.log.Print("if else begin ", v.incLevel())
		ast.Walk(v, t)
		v.log.Print("if else end ", v.decLevel())
	}

	return nil
}

func (v *complexityVisitor) visitSwitchStmt(n *ast.SwitchStmt) ast.Visitor {
	if n := n.Init; n != nil {
		v.log.Print("switch init begin", typeName(n))
		ast.Walk(v, n)
		v.log.Print("switch init end", typeName(n))
	}

	if tag := n.Tag; tag != nil {
		v.nestIncComplexity()
		v.log.Print("switch tag begin", typeName(tag))
		ast.Walk(v, tag)
		v.log.Print("switch tag end", typeName(tag))

		v.incNesting()
		v.log.Print("switch tag body begin")
		for i, tmp := range n.Body.List {
			v.log.Printf("switch tag case begin. *%d*", i)

			n, _ := tmp.(*ast.CaseClause)
			for _, n := range n.Body {
				ast.Walk(v, n)
			}

			v.log.Printf("switch tag case end. *%d*", i)
		}
		v.log.Print("switch tag body end")
		v.decNesting()
		return nil
	}

	if len(n.Body.List) == 0 {
		v.log.Print("switch body is empty")
		return nil
	}

	v.log.Print("switch body begin")
	for i, tmp := range n.Body.List {
		v.log.Printf("switch case begin. *%d*", i)

		if i == 0 {
			v.nestIncComplexity()
		} else {
			v.incComplexity()
		}

		n, _ := tmp.(*ast.CaseClause)
		for _, expr := range n.List {
			ast.Walk(v, expr)
		}

		v.incNesting()
		for _, n := range n.Body {
			ast.Walk(v, n)
		}
		v.decNesting()

		v.log.Printf("switch case end. *%d*", i)
	}
	v.log.Print("switch body end")
	return nil
}

func (v *complexityVisitor) visitTypeSwitchStmt(n *ast.TypeSwitchStmt) ast.Visitor {
	v.nestIncComplexity()

	if n := n.Init; n != nil {
		v.log.Print("switch type init begin")
		ast.Walk(v, n)
		v.log.Print("switch type init end")
	}

	if n := n.Assign; n != nil {
		v.log.Print("switch type assign begin", typeName(n))
		ast.Walk(v, n)
		v.log.Print("switch type assign end", typeName(n))
	}

	v.incNesting()
	v.log.Print("switch type body begin")
	ast.Walk(v, n.Body)
	v.log.Print("switch type body end")
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

func (v *complexityVisitor) visitBinaryExpr(n *ast.BinaryExpr) ast.Visitor {
	if v.isCalculated(n) {
		ast.Walk(v, n.X)
		ast.Walk(v, n.Y)
		return nil
	}

	// v.printBody(n)
	ops := v.collectBinaryOps(n)

	var lastOp token.Token
	cache := make([]token.Token, 0, len(ops)/3)
	for _, op := range ops {
		v.log.Printf("op: %s", op.String())
		switch op {
		default:
			// v.log.Printf("xxx op skip: %s", op.String())

		case token.LPAREN:
			// v.log.Printf("xxx op paren %d op: %s, last: %s", len(cache), op.String(), lastOp.String())
			cache = append(cache, lastOp)
			lastOp = op

		case token.RPAREN:
			lastOp = cache[len(cache)-1]
			cache = cache[:len(cache)-1]
			// v.log.Printf("xxx op paren %d op: %s, last: %s", len(cache), op.String(), lastOp.String())

		case token.LAND, token.LOR:
			// v.log.Printf("xxx op: %s, last: %s", op.String(), lastOp.String())
			if lastOp != op {
				v.incComplexity()
			}
			lastOp = op
		}
	}

	ast.Walk(v, n.X)
	ast.Walk(v, n.Y)
	return nil
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
		ops := v.collectBinaryOps(exp.X)
		res := make([]token.Token, 0, len(ops)+2)
		res = append(res, token.LPAREN)
		res = append(res, ops...)
		res = append(res, token.RPAREN)
		return res

	case *ast.UnaryExpr:
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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
func (ioDiscard) WriteString(s string) (int, error) {
	return len(s), nil
}
