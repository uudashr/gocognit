package e

// foo keeps the call-argument case below honest: a logical sequence inside a
// call argument is a separate sequence from the surrounding expression.
func foo(x bool) bool { return x }

func LogicalSeqFlat(a, b, c bool) { // want "cognitive complexity 2 of func LogicalSeqFlat is high \\(> 0\\)"
	if a && b && c { // +1 for `if`, +1 for the `&&` sequence
	}
} // total complexity = 2

func LogicalSeqParens(a, b, c bool) { // want "cognitive complexity 2 of func LogicalSeqParens is high \\(> 0\\)"
	if a && (b && c) { // +1 for `if`, +1 for the `&&` sequence; redundant parens are transparent
	}
} // total complexity = 2

func LogicalSeqCallArg(a, b, c bool) bool { // want "cognitive complexity 3 of func LogicalSeqCallArg is high \\(> 0\\)"
	return a && b || foo(b && c) // +1 `&&`, +1 `||`, +1 for the `&&` inside the call argument
} // total complexity = 3

func LogicalSeqParenThenOp(a, b, c, d bool) bool { // want "cognitive complexity 2 of func LogicalSeqParenThenOp is high \\(> 0\\)"
	return a && (b || c) || d // +1 `&&`, +1 `||` sequence; the parens do not start a new one
} // total complexity = 2

func LogicalSeqParenInsideIf(a, b, c bool) { // want "cognitive complexity 2 of func LogicalSeqParenInsideIf is high \\(> 0\\)"
	if a || (b || c) { // +1 for `if`, +1 for the `||` sequence
	}
} // total complexity = 2

func LogicalSeqNegatedGroup(a, b, c bool) { // want "cognitive complexity 3 of func LogicalSeqNegatedGroup is high \\(> 0\\)"
	if a && !(b && c) { // +1 for `if`, +1 for outer `&&`, +1 for the `&&` inside the negated group
	}
} // total complexity = 3

func LogicalSeqMixed(a, b, c, d, e bool) { // want "cognitive complexity 5 of func LogicalSeqMixed is high \\(> 0\\)"
	if a && b || c && d || e { // +1 for `if`, +4 for the `&&` `||` `&&` `||` sequences
	}
} // total complexity = 5

func LogicalSeqLongAnd(a, b, c, d, e bool) { // want "cognitive complexity 2 of func LogicalSeqLongAnd is high \\(> 0\\)"
	if a && b && c && d && e { // +1 for `if`, +1 for the single `&&` sequence
	}
} // total complexity = 2

func LogicalSeqNested(a, b, c, d, e, f, g, h, i, j, k, l, m bool) { // want "cognitive complexity 7 of func LogicalSeqNested is high \\(> 0\\)"
	if a && b && c ||
		d ||
		e && f && g ||
		(h ||
			(i && j || k)) ||
		l ||
		m {
	}
} // total complexity = 7
