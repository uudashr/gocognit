package gocognit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestRecursiveNodes exercises the Tarjan core directly. Only pointer identity
// matters, so the nodes are synthetic declarations; names are deliberately all
// the same to prove the algorithm keys on identity rather than name.
func TestRecursiveNodes(t *testing.T) {
	tests := []struct {
		name      string
		nodeCount int
		edges     [][2]int // node index -> node index
		want      []int    // node indices expected to be recursive
	}{
		{name: "empty", nodeCount: 0},
		{name: "isolated", nodeCount: 1},
		{name: "self loop", nodeCount: 1, edges: [][2]int{{0, 0}}, want: []int{0}},
		{name: "pair", nodeCount: 2, edges: [][2]int{{0, 1}, {1, 0}}, want: []int{0, 1}},
		{name: "triple", nodeCount: 3, edges: [][2]int{{0, 1}, {1, 2}, {2, 0}}, want: []int{0, 1, 2}},
		{name: "tail into self loop", nodeCount: 2, edges: [][2]int{{0, 1}, {1, 1}}, want: []int{1}},
		{name: "tail into pair", nodeCount: 3, edges: [][2]int{{0, 1}, {1, 2}, {2, 1}}, want: []int{1, 2}},
		{name: "diamond is acyclic", nodeCount: 4, edges: [][2]int{{0, 1}, {0, 2}, {1, 3}, {2, 3}}},
		{name: "exit from cycle", nodeCount: 3, edges: [][2]int{{0, 1}, {1, 0}, {1, 2}}, want: []int{0, 1}},
		{name: "two disjoint cycles", nodeCount: 5, edges: [][2]int{{0, 1}, {1, 0}, {2, 3}, {3, 2}}, want: []int{0, 1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := make([]*ast.FuncDecl, tt.nodeCount)
			adjacency := make(map[*ast.FuncDecl][]*ast.FuncDecl, tt.nodeCount)

			for i := range nodes {
				nodes[i] = &ast.FuncDecl{Name: ast.NewIdent("f")}
				adjacency[nodes[i]] = nil
			}

			for _, e := range tt.edges {
				adjacency[nodes[e[0]]] = append(adjacency[nodes[e[0]]], nodes[e[1]])
			}

			want := make(map[*ast.FuncDecl]bool, len(tt.want))
			for _, i := range tt.want {
				want[nodes[i]] = true
			}

			got := recursiveNodes(nodes, adjacency)

			for i, n := range nodes {
				if got[n] != want[n] {
					t.Errorf("node %d: recursive = %v, want %v", i, got[n], want[n])
				}
			}
		})
	}
}

// TestScanComplexityRecursionIncrement locks the per-cycle, flat contract: two
// recursive call sites produce exactly one increment, with Nesting 0, and it is
// attributed to the function declaration.
func TestScanComplexityRecursionIncrement(t *testing.T) {
	const src = `package p

func Fib(n int) int {
	if n <= 1 {
		return n
	}

	return Fib(n-1) + Fib(n-2)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatal("expected a FuncDecl")
	}

	res := ScanComplexityWithRecursion(fn, true, true)
	if res.Complexity != 2 {
		t.Errorf("Complexity = %d, want 2 (if + one recursion)", res.Complexity)
	}

	increments := 0
	for _, d := range res.Diagnostics {
		if d.Text != "recursion" {
			continue
		}

		increments++
		if d.Inc != 1 || d.Nesting != 0 {
			t.Errorf("recursion diagnostic = {Inc:%d Nesting:%d}, want {1 0}", d.Inc, d.Nesting)
		}

		if d.Pos != fn.Pos() {
			t.Errorf("recursion diagnostic at %v, want declaration position %v", d.Pos, fn.Pos())
		}
	}

	if increments != 1 {
		t.Errorf("recursion increments = %d, want 1", increments)
	}

	if got := ScanComplexityWithRecursion(fn, false, false).Complexity; got != 1 {
		t.Errorf("non-recursive Complexity = %d, want 1", got)
	}
}

// TestPossibleMethodRecursion covers the low-noise heuristic that decides
// whether to suggest type-aware analysis. It must fire on mutual method cycles
// (including across files) and stay quiet for resolved recursion, acyclic
// method calls, and imported selectors.
func TestPossibleMethodRecursion(t *testing.T) {
	tests := []struct {
		name    string
		sources []string
		want    bool
	}{
		{
			name: "mutual methods",
			sources: []string{`package p

type Node struct{ next *Node }

func (n *Node) Walk() { n.Step() }
func (n *Node) Step() { n.next.Walk() }
`},
			want: true,
		},
		{
			name: "cross-file mutual methods",
			sources: []string{
				"package p\n\ntype Node struct{ next *Node }\n\nfunc (n *Node) Walk() { n.Step() }\n",
				"package p\n\nfunc (n *Node) Step() { n.next.Walk() }\n",
			},
			want: true,
		},
		{
			name: "direct recursion is already resolved",
			sources: []string{`package p

func Fib(n int) int {
	if n <= 1 {
		return n
	}
	return Fib(n-1) + Fib(n-2)
}
`},
		},
		{
			name: "sibling method is not a cycle",
			sources: []string{`package p

type Node struct{}

func (n *Node) A() { n.B() }
func (n *Node) B()   {}
`},
		},
		{
			name: "method and function cycle",
			sources: []string{`package p

type Node struct{}

func (n *Node) Walk()  { stepInto(n) }
func stepInto(n *Node) { n.Walk() }
`},
			want: true,
		},
		{
			name: "imported selector is ignored",
			sources: []string{`package p

import "fmt"

type T struct{}

func (t *T) Println() {}
func (t *T) Use()     { fmt.Println("x") }
`},
		},
		{
			name: "foreign String call is not a cycle",
			sources: []string{`package p

type T struct{}

type builder struct{}

func (b *builder) String() string { return "" }
func (t *T) String() string       { return "" }
func (t *T) Render(b *builder)    { _ = b.String() }
`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			files := make([]*ast.File, 0, len(tt.sources))

			for i, src := range tt.sources {
				f, err := parser.ParseFile(fset, fmt.Sprintf("src%d.go", i), src, 0)
				if err != nil {
					t.Fatal(err)
				}

				files = append(files, f)
			}

			if got := PossibleMethodRecursion(files); got != tt.want {
				t.Errorf("PossibleMethodRecursion() = %v, want %v", got, tt.want)
			}
		})
	}
}
