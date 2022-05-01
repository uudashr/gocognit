//go:build go1.18
// +build go1.18

package gocognit

import (
	"go/ast"
	"strings"
)

// recvString returns a string representation of recv of the
// form "T", "*T", Type[T], Type[T, V], or "BADRECV" (if not a proper receiver type).
func recvString(recv ast.Expr) string {
	switch t := recv.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + recvString(t.X)
	case *ast.IndexExpr:
		return recvString(t.X) + "[" + recvString(t.Index) + "]"
	case *ast.IndexListExpr:
		targs := make([]string, len(t.Indices))
		for i, exp := range t.Indices {
			targs[i] = recvString(exp)
		}

		return recvString(t.X) + "[" + strings.Join(targs, ", ") + "]"
	}
	return "BADRECV"
}
