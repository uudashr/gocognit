package gocognit

import (
	"go/ast"
	"go/types"
)

// RecursiveFuncs reports which top-level functions and methods in files take
// part in a recursion cycle, whether direct or indirect.
//
// A function is reported when it lies on a cycle of mutually recursive calls
// (a strongly connected component with more than one member) or when it calls
// itself directly. All declarations of the given files are expected to belong
// to the same package.
//
// When info is non-nil it is used to resolve calls with type information. That
// covers methods and references across files of the same package. When info is
// nil a syntactic approximation is used: package-level functions and direct
// calls through a method's own receiver are recognized, but mutual method
// recursion may go undetected. Calls into other packages are never resolved.
//
// See also [ScanComplexityWithRecursion], which scores a declaration with the
// result, and [ComplexityStatsForFiles], which builds the statistics for a
// whole package.
func RecursiveFuncs(files []*ast.File, info *types.Info) map[*ast.FuncDecl]bool {
	var (
		decls  []*ast.FuncDecl
		byName = make(map[string]*ast.FuncDecl)
		byObj  = make(map[*types.Func]*ast.FuncDecl)
	)

	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			decls = append(decls, fn)

			if info != nil {
				if obj, ok := info.Defs[fn.Name].(*types.Func); ok {
					byObj[obj] = fn
				}
			} else if fn.Recv == nil {
				byName[fn.Name.Name] = fn
			}
		}
	}

	if len(decls) == 0 {
		return nil
	}

	adjacency := make(map[*ast.FuncDecl][]*ast.FuncDecl, len(decls))
	for _, fn := range decls {
		adjacency[fn] = callees(fn, info, byName, byObj)
	}

	return recursiveNodes(decls, adjacency)
}

// directRecursive reports whether fn calls itself. It is the recursion
// detection available to the single-function API, which has no package
// context; [RecursiveFuncs] is used when the whole package is available. Both
// share the same call resolver, so direct self-calls are recognized in the
// same shapes (plain calls, generic instantiations and calls through a
// method's own receiver).
func directRecursive(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	byName := make(map[string]*ast.FuncDecl)
	if fn.Recv == nil {
		byName[fn.Name.Name] = fn
	}

	for _, callee := range callees(fn, nil, byName, nil) {
		if callee == fn {
			return true
		}
	}

	return false
}

// callees returns the package-local functions and methods called from the body
// of fn.
func callees(fn *ast.FuncDecl, info *types.Info, byName map[string]*ast.FuncDecl, byObj map[*types.Func]*ast.FuncDecl) []*ast.FuncDecl {
	var out []*ast.FuncDecl

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if target := resolveCall(call.Fun, fn, info, byName, byObj); target != nil {
			out = append(out, target)
		}

		return true
	})

	return out
}

// resolveCall resolves the function or method invoked by fun, returning nil
// when it is not a declaration of the analyzed package.
func resolveCall(fun ast.Expr, current *ast.FuncDecl, info *types.Info, byName map[string]*ast.FuncDecl, byObj map[*types.Func]*ast.FuncDecl) *ast.FuncDecl {
	switch f := fun.(type) {
	case *ast.ParenExpr:
		return resolveCall(f.X, current, info, byName, byObj)
	case *ast.IndexExpr: // generic instantiation, e.g. F[int](x)
		return resolveCall(f.X, current, info, byName, byObj)
	case *ast.IndexListExpr:
		return resolveCall(f.X, current, info, byName, byObj)
	case *ast.Ident:
		if info != nil {
			if obj, ok := info.Uses[f].(*types.Func); ok {
				return byObj[obj]
			}

			return nil
		}

		if f.Obj != nil {
			if fd, ok := f.Obj.Decl.(*ast.FuncDecl); ok && fd.Body != nil {
				return fd
			}

			return nil
		}

		return byName[f.Name]
	case *ast.SelectorExpr:
		if info != nil {
			if obj, ok := info.Uses[f.Sel].(*types.Func); ok {
				return byObj[obj]
			}

			return nil
		}

		// Syntactic fallback: recognize a call through the current method's
		// own receiver, e.g. (n *Node) Walk calling n.Walk.
		if current.Recv != nil && f.Sel.Name == current.Name.Name {
			if id, ok := f.X.(*ast.Ident); ok && isReceiverName(current, id.Name) {
				return current
			}
		}

		return nil
	}

	return nil
}

func isReceiverName(fn *ast.FuncDecl, name string) bool {
	for _, field := range fn.Recv.List {
		for _, id := range field.Names {
			if id.Name == name {
				return true
			}
		}
	}

	return false
}

// recursiveNodes marks every node that lies on a cycle of adjacency: either a
// self-loop or a strongly connected component with more than one node. It uses
// an iterative Tarjan traversal so that deeply nested call graphs cannot
// overflow the goroutine stack.
func recursiveNodes(nodes []*ast.FuncDecl, adjacency map[*ast.FuncDecl][]*ast.FuncDecl) map[*ast.FuncDecl]bool {
	type frame struct {
		node *ast.FuncDecl
		next int
	}

	var (
		index     int
		indices   = make(map[*ast.FuncDecl]int, len(nodes))
		lowlink   = make(map[*ast.FuncDecl]int, len(nodes))
		onStack   = make(map[*ast.FuncDecl]bool, len(nodes))
		stack     []*ast.FuncDecl
		recursive = make(map[*ast.FuncDecl]bool)
	)

	for _, root := range nodes {
		if _, visited := indices[root]; visited {
			continue
		}

		indices[root] = index
		lowlink[root] = index
		index++
		stack = append(stack, root)
		onStack[root] = true

		work := []frame{{node: root}}

		for len(work) > 0 {
			top := &work[len(work)-1]
			v := top.node

			if top.next < len(adjacency[v]) {
				w := adjacency[v][top.next]
				top.next++

				if _, visited := indices[w]; !visited {
					indices[w] = index
					lowlink[w] = index
					index++
					stack = append(stack, w)
					onStack[w] = true
					work = append(work, frame{node: w})
				} else if onStack[w] && indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}

				continue
			}

			if lowlink[v] == indices[v] {
				var component []*ast.FuncDecl
				for {
					w := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					onStack[w] = false
					component = append(component, w)
					if w == v {
						break
					}
				}

				if len(component) > 1 {
					for _, w := range component {
						recursive[w] = true
					}
				}
			}

			work = work[:len(work)-1]
			if len(work) > 0 {
				parent := work[len(work)-1].node
				if lowlink[v] < lowlink[parent] {
					lowlink[parent] = lowlink[v]
				}
			}
		}
	}

	// A self-loop is a recursion cycle that contains a single method.
	for _, v := range nodes {
		for _, w := range adjacency[v] {
			if w == v {
				recursive[v] = true
				break
			}
		}
	}

	if len(recursive) == 0 {
		return nil
	}

	return recursive
}
