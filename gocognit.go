package gocognit

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Stat is statistic of the complexity.
type Stat struct {
	PkgName     string
	FuncName    string
	Complexity  int
	Pos         token.Position
	Diagnostics []Diagnostic `json:",omitempty"`
}

// Diagnostic contains information how the complexity increase.
type Diagnostic struct {
	Inc     int
	Nesting int `json:",omitempty"`
	Text    string
	Pos     DiagnosticPosition
}

// DiagnosticPosition is the position of the diagnostic.
type DiagnosticPosition struct {
	Offset int // offset, starting at 0
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (byte count)
}

func (pos DiagnosticPosition) isValid() bool {
	return pos.Line > 0
}

func (pos DiagnosticPosition) String() string {
	var s string
	if pos.isValid() {
		if s != "" {
			s += ":"
		}

		s += strconv.Itoa(pos.Line)
		if pos.Column != 0 {
			s += fmt.Sprintf(":%d", pos.Column)
		}
	}

	if s == "" {
		s = "-"
	}

	return s
}

func (d Diagnostic) String() string {
	if d.Nesting == 0 {
		return fmt.Sprintf("+%d", d.Inc)
	}

	return fmt.Sprintf("+%d (nesting=%d)", d.Inc, d.Nesting)
}

func (s Stat) String() string {
	return fmt.Sprintf("%d %s %s %s", s.Complexity, s.PkgName, s.FuncName, s.Pos)
}

// ComplexityOptions controls complexity calculation options.
type ComplexityOptions struct {
	Diagnostics       bool
	IgnoreErrorChecks bool
}

// ComplexityStats builds the complexity statistics.
func ComplexityStats(f *ast.File, fset *token.FileSet, stats []Stat) []Stat {
	return ComplexityStatsWithOptions(f, fset, stats, ComplexityOptions{})
}

// ComplexityStatsWithDiagnostic builds the complexity statistics with diagnostic.
func ComplexityStatsWithDiagnostic(f *ast.File, fset *token.FileSet, stats []Stat, enableDiagnostics bool) []Stat {
	return ComplexityStatsWithOptions(f, fset, stats, ComplexityOptions{
		Diagnostics: enableDiagnostics,
	})
}

// ComplexityStatsWithOptions builds the complexity statistics with options.
func ComplexityStatsWithOptions(f *ast.File, fset *token.FileSet, stats []Stat, opts ComplexityOptions) []Stat {
	return ComplexityStatsForFilesWithOptions([]*ast.File{f}, fset, nil, stats, opts)
}

// ComplexityStatsForFiles builds the complexity statistics for a package made
// up of the given files. Recursion is detected across the whole set, so
// indirect recursion is counted. When info is non-nil it is used to resolve
// calls precisely; otherwise a syntactic approximation is used.
func ComplexityStatsForFiles(files []*ast.File, fset *token.FileSet, info *types.Info, stats []Stat) []Stat {
	return ComplexityStatsForFilesWithOptions(files, fset, info, stats, ComplexityOptions{})
}

// ComplexityStatsForFilesWithDiagnostic builds the complexity statistics for a
// package made up of the given files, optionally with diagnostic output.
func ComplexityStatsForFilesWithDiagnostic(files []*ast.File, fset *token.FileSet, info *types.Info, stats []Stat, enableDiagnostics bool) []Stat {
	return ComplexityStatsForFilesWithOptions(files, fset, info, stats, ComplexityOptions{
		Diagnostics: enableDiagnostics,
	})
}

// ComplexityStatsForFilesWithOptions builds the complexity statistics for a
// package made up of the given files, with options.
func ComplexityStatsForFilesWithOptions(files []*ast.File, fset *token.FileSet, info *types.Info, stats []Stat, opts ComplexityOptions) []Stat {
	recursive := RecursiveFuncs(files, info)

	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			d := parseDirective(fn.Doc)
			if d.Ignore {
				continue
			}

			res := ScanComplexityWithRecursionAndOptions(fn, recursive[fn], opts)

			stats = append(stats, Stat{
				PkgName:     f.Name.Name,
				FuncName:    funcName(fn),
				Complexity:  res.Complexity,
				Diagnostics: generateDiagnostics(fset, res.Diagnostics),
				Pos:         fset.Position(fn.Pos()),
			})
		}
	}

	return stats
}

func generateDiagnostics(fset *token.FileSet, diags []diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(diags))

	for _, diag := range diags {
		pos := fset.Position(diag.Pos)
		diagPos := DiagnosticPosition{
			Offset: pos.Offset,
			Line:   pos.Line,
			Column: pos.Column,
		}

		out = append(out, Diagnostic{
			Inc:     diag.Inc,
			Nesting: diag.Nesting,
			Text:    diag.Text,
			Pos:     diagPos,
		})
	}

	return out
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
	return ComplexityWithOptions(fn, ComplexityOptions{})
}

// ComplexityWithOptions calculates the cognitive complexity of a function with options.
func ComplexityWithOptions(fn *ast.FuncDecl, opts ComplexityOptions) int {
	res := ScanComplexityWithOptions(fn, opts)

	return res.Complexity
}

// ScanComplexity scans the function declaration. Recursion is detected for
// direct self-calls only: a lone declaration carries no information about the
// surrounding package, so indirect cycles cannot be seen. Prefer a
// package-aware entry point such as [ComplexityStatsForFiles], or combine
// [RecursiveFuncs] with [ScanComplexityWithRecursion].
func ScanComplexity(fn *ast.FuncDecl, includeDiagnostics bool) ScanResult {
	return ScanComplexityWithOptions(fn, ComplexityOptions{
		Diagnostics: includeDiagnostics,
	})
}

// ScanComplexityWithOptions scans the function declaration with options.
func ScanComplexityWithOptions(fn *ast.FuncDecl, opts ComplexityOptions) ScanResult {
	return scanComplexity(fn, scanOptions{
		includeDiagnostics: opts.Diagnostics,
		ignoreErrorChecks:  opts.IgnoreErrorChecks,
		recursive:          directRecursive(fn),
	})
}

// ScanComplexityWithRecursion scans the function declaration with recursion
// information from a package-level analysis (see [RecursiveFuncs]). When
// recursive is true, exactly one fundamental increment is added for the
// recursion cycle, regardless of how many recursive calls the function makes.
func ScanComplexityWithRecursion(fn *ast.FuncDecl, recursive bool, includeDiagnostics bool) ScanResult {
	return ScanComplexityWithRecursionAndOptions(fn, recursive, ComplexityOptions{
		Diagnostics: includeDiagnostics,
	})
}

// ScanComplexityWithRecursionAndOptions scans the function declaration with
// recursion information from a package-level analysis and options.
func ScanComplexityWithRecursionAndOptions(fn *ast.FuncDecl, recursive bool, opts ComplexityOptions) ScanResult {
	return scanComplexity(fn, scanOptions{
		includeDiagnostics: opts.Diagnostics,
		ignoreErrorChecks:  opts.IgnoreErrorChecks,
		recursive:          recursive,
	})
}

// scanOptions controls how scanComplexity accounts for recursion and error checking.
type scanOptions struct {
	includeDiagnostics bool
	ignoreErrorChecks  bool

	// recursive reports whether fn takes part in a recursion cycle. The
	// increment is a property of the function, not of each call, so it is
	// applied once after the walk.
	recursive bool
}

func scanComplexity(fn *ast.FuncDecl, opts scanOptions) ScanResult {
	v := complexityVisitor{
		name:               fn.Name,
		diagnosticsEnabled: opts.includeDiagnostics,
		ignoreErrorChecks:  opts.ignoreErrorChecks,
	}

	ast.Walk(&v, fn)

	if opts.recursive {
		v.incComplexity("recursion", fn.Pos())
	}

	return ScanResult{
		Diagnostics: v.diagnostics,
		Complexity:  v.complexity,
	}
}

type ScanResult struct {
	Diagnostics []diagnostic
	Complexity  int
}

type diagnostic struct {
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

	diagnosticsEnabled bool
	diagnostics        []diagnostic

	ignoreErrorChecks bool
}

func (v *complexityVisitor) incNesting() {
	v.nesting++
}

func (v *complexityVisitor) decNesting() {
	v.nesting--
}

func (v *complexityVisitor) incComplexity(text string, pos token.Pos) {
	v.complexity++

	if !v.diagnosticsEnabled {
		return
	}

	v.diagnostics = append(v.diagnostics, diagnostic{
		Inc:  1,
		Text: text,
		Pos:  pos,
	})
}

func (v *complexityVisitor) nestIncComplexity(text string, pos token.Pos) {
	v.complexity += (v.nesting + 1)

	if !v.diagnosticsEnabled {
		return
	}

	v.diagnostics = append(v.diagnostics, diagnostic{
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
	}

	return v
}

func (v *complexityVisitor) visitIfStmt(n *ast.IfStmt) ast.Visitor {
	if v.ignoreErrorChecks && v.isIdiomaticErrorCheck(n) {
		if n := n.Init; n != nil {
			ast.Walk(v, n)
		}

		ast.Walk(v, n.Cond)

		v.incNesting()
		ast.Walk(v, n.Body)
		v.decNesting()

		return nil
	}

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

		v.incNesting()
		ast.Walk(v, n.Else)
		v.decNesting()
	} else if _, ok := n.Else.(*ast.IfStmt); ok {
		v.markAsElseNode(n.Else)
		ast.Walk(v, n.Else)
	}

	return nil
}

func (v *complexityVisitor) isIdiomaticErrorCheck(n *ast.IfStmt) bool {
	if n.Else != nil || v.markedAsElseNode(n) {
		return false
	}

	return isErrCheckCond(n.Cond)
}

func isErrCheckCond(cond ast.Expr) bool {
	cond = unwrapParen(cond)
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}

	x := unwrapParen(bin.X)
	y := unwrapParen(bin.Y)

	return (isErrIdent(x) && isNilIdent(y)) || (isNilIdent(x) && isErrIdent(y))
}

func unwrapParen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func isErrIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	name := ident.Name
	if name == "err" {
		return true
	}
	if strings.HasPrefix(name, "err") || strings.HasSuffix(name, "Err") || strings.HasSuffix(name, "_err") {
		return true
	}
	return false
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

func (v *complexityVisitor) collectBinaryOps(exp ast.Expr) []token.Token {
	v.markCalculated(exp)

	if paren, ok := exp.(*ast.ParenExpr); ok {
		return v.collectBinaryOps(paren.X)
	}

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
	over              int  // -over flag
	ignoreErrorChecks bool // -ignore-error-checks flag
)

func init() {
	Analyzer.Flags.IntVar(&over, "over", over, "show functions with complexity > N only")
	Analyzer.Flags.BoolVar(&ignoreErrorChecks, "ignore-error-checks", false, "ignore idiomatic error checks")
	Analyzer.Flags.BoolVar(&ignoreErrorChecks, "ignore-err", false, "ignore idiomatic error checks (shorthand)")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	recursive := RecursiveFuncs(pass.Files, pass.TypesInfo)

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

		fnComplexity := ScanComplexityWithRecursionAndOptions(funcDecl, recursive[funcDecl], ComplexityOptions{
			IgnoreErrorChecks: ignoreErrorChecks,
		}).Complexity

		if fnComplexity > over {
			pass.Reportf(funcDecl.Pos(), "cognitive complexity %d of func %s is high (> %d)", fnComplexity, fnName, over)
		}
	})

	return nil, nil
}
