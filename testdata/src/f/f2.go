package f

func MutualB(n int) int { // want "cognitive complexity 1 of func MutualB is high \\(> 0\\)"
	return MutualA(n) // +1 for the recursion cycle
} // total complexity = 1

func Cycle2(n int) int { // want "cognitive complexity 1 of func Cycle2 is high \\(> 0\\)"
	return Cycle3(n - 1) // +1 for the recursion cycle
} // total complexity = 1

func Cycle3(n int) int { // want "cognitive complexity 1 of func Cycle3 is high \\(> 0\\)"
	return Cycle1(n - 1) // +1 for the recursion cycle
} // total complexity = 1

type Node struct{ next *Node }

// Walk and Step are mutually recursive methods on the same receiver type.
func (n *Node) Walk() { // want "cognitive complexity 2 of func \\(\\*Node\\)\\.Walk is high \\(> 0\\)"
	for n != nil { // +1
		n.Step() // +1 for the recursion cycle
		n = n.next
	}
} // total complexity = 2

func (n *Node) Step() { // want "cognitive complexity 1 of func \\(\\*Node\\)\\.Step is high \\(> 0\\)"
	n.next.Walk() // +1 for the recursion cycle
} // total complexity = 1

// Loop is a direct method self-call.
func (n *Node) Loop() { // want "cognitive complexity 1 of func \\(\\*Node\\)\\.Loop is high \\(> 0\\)"
	n.Loop() // +1 for the recursion cycle
} // total complexity = 1
