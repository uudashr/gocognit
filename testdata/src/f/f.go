package f

// Fib calls itself twice but has a single recursion cycle, so it earns one
// flat increment, not one per call site.
func Fib(n int) int { // want "cognitive complexity 2 of func Fib is high \\(> 0\\)"
	if n <= 1 { // +1
		return n
	}

	return Fib(n-1) + Fib(n-2) // +1 for the recursion cycle
} // total complexity = 2

// MutualA and MutualB (f2.go) form an indirect recursion cycle: each method in
// the cycle gets its own increment.
func MutualA(n int) int { // want "cognitive complexity 2 of func MutualA is high \\(> 0\\)"
	if n <= 0 { // +1
		return 0
	}

	return MutualB(n - 1) // +1 for the recursion cycle
} // total complexity = 2

// Cycle1, Cycle2 and Cycle3 (f2.go) form a three-method cycle.
func Cycle1(n int) int { // want "cognitive complexity 2 of func Cycle1 is high \\(> 0\\)"
	if n <= 0 { // +1
		return 0
	}

	return Cycle2(n - 1) // +1 for the recursion cycle
} // total complexity = 2

// notRecursive calls helper, which never calls back, so no increment is added.
func notRecursive(n int) int { // want "cognitive complexity 1 of func notRecursive is high \\(> 0\\)"
	if n <= 0 { // +1
		return 0
	}

	return helper(n - 1) // not part of a cycle
} // total complexity = 1

// CallerCallsCycle calls into a cycle but is not itself part of one, so it
// gets no recursion increment.
func CallerCallsCycle(n int) int { // want "cognitive complexity 1 of func CallerCallsCycle is high \\(> 0\\)"
	if n <= 0 { // +1
		return 0
	}

	return Cycle1(n) // no increment: CallerCallsCycle is not in the cycle
} // total complexity = 1

func helper(n int) int { return n } // total complexity = 0

// Sum is a generic function that calls itself.
func Sum[T int | int64](n T, xs []T) T { // want "cognitive complexity 2 of func Sum is high \\(> 0\\)"
	if len(xs) == 0 { // +1
		return n
	}

	return Sum(n+xs[0], xs[1:]) // +1 for the recursion cycle
} // total complexity = 2
