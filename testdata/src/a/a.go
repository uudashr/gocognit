package testdata

import (
	"errors"
	"fmt"
	"io"
)

func HelloWorld() string {
	_ = len("hello")
	return "Hello, World!"
} // total complexity = 0

func SimpleCond(n int) string { // want "cognitive complexity 1 of func SimpleCond is high \\(> 0\\)"
	if n == 100 { // +1
		return "a hundred"
	}

	return "others"
} // total complexity = 1

func IfElseNested(n int) string { // want "cognitive complexity 3 of func IfElseNested is high \\(> 0\\)"
	if n == 100 { // +1
		return "a hundred"
	} else { // + 1
		if n == 200 { // + 1
			return "two hundred"
		}
	}

	return "others"
} // total complexity = 3

func IfElseIfNested(n int) string { // want "cognitive complexity 3 of func IfElseIfNested is high \\(> 0\\)"
	if n == 100 { // +1
		return "a hundred"
	} else if n < 300 { // + 1
		if n == 200 { // + 1
			return "two hundred"
		}
	}

	return "others"
} // total complexity = 3

func SimpleLogicalSeq1(a, b, c, d bool) string { // want "cognitive complexity 2 of func SimpleLogicalSeq1 is high \\(> 0\\)"
	if a && b && c && d { // +1 for `if`, +1 for `&&` sequence
		return "ok"
	}

	return "not ok"
} // total complexity = 2

func SimpleLogicalSeq2(a, b, c, d bool) string { // want "cognitive complexity 2 of func SimpleLogicalSeq2 is high \\(> 0\\)"
	if a || b || c || d { // +1 for `if`, +1 for `||` sequence
		return "ok"
	}

	return "not ok"
} // total complexity = 2

func SimpleLogicalSeq3(a, b, c interface{}) string { // want "cognitive complexity 2 of func SimpleLogicalSeq3 is high \\(> 0\\)"
	if a == nil || b != nil || c != nil { // +1 for `if`, +1 for `||` sequence
		return "ok"
	}

	return "not ok"
} // total complexity = 2

func ComplexLogicalSeq1(a, b, c, d, e, f bool) string { // want "cognitive complexity 4 of func ComplexLogicalSeq1 is high \\(> 0\\)"
	if a && b && c || d || e && f { // +1 for `if`, +3 for changing sequence of `&&` `||` `&&`
		return "ok"
	}

	return "not ok"
} // total complexity = 4

func ComplexLogicalSeq2(a, b, c, d, e, f bool) string { // want "cognitive complexity 3 of func ComplexLogicalSeq2 is high \\(> 0\\)"
	if a && !(b && c) { // +1 for `if`, +1 for each `&&` chain
		return "ok"
	}

	return "not ok"
} // total complexity = 3

func ComplexLogicalSeq3(a, b, c, d, e, f bool) string { // want "cognitive complexity 3 of func ComplexLogicalSeq3 is high \\(> 0\\)"
	if a && (b && c) { // +1 for `if`, +1 for each `&&` chain
		return "ok"
	}

	return "not ok"
} // total complexity = 3

func ComplexLogicalSeq4(a, b, c, d, e, f bool) bool { // want "cognitive complexity 3 of func ComplexLogicalSeq4 is high \\(> 0\\)"
	return a && b && c || d || e && f // +3 for changing sequence of `&&` `||` `&&`
} // total complexity = 3

func ComplexLogicalSeq5(a, b, c, d, e, f bool) bool { // want "cognitive complexity 3 of func ComplexLogicalSeq5 is high \\(> 0\\)"
	return a && b && (c && d || e || f) // +1 for `&&` sequence, +2 for `&&` `||` sequence in parentheses
} // total complexity = 3

func ExprFunc(a, b, c interface{}) bool { // want "cognitive complexity 2 of func ExprFunc is high \\(> 0\\)"
	if a != nil || b != nil || c != nil { // +1 for `if`, +1 for `||` chain
		return false
	}

	return true
} // total complexity = 2

func VarFunc(a, b, c interface{}) bool { // want "cognitive complexity 2 of func VarFunc is high \\(> 0\\)"
	na := a != nil
	nb := b != nil
	nc := c != nil
	if na || nb || nc { // +1 for `if`, +1 for `||` chain
		return false
	}

	return true
} // total complexity = 2

func GetWords(number int) string { // want "cognitive complexity 1 of func GetWords is high \\(> 0\\)"
	switch number { // +1
	case 1:
		return "one"
	case 2:
		return "a couple"
	case 3:
		return "a few"
	default:
		return "lots"
	}
} // Cognitive complexity = 1

func GetWords_Complex(number int) string { // want "cognitive complexity 4 of func GetWords_Complex is high \\(> 0\\)"
	if number == 1 { // +1
		return "one"
	} else if number == 2 { // +1
		return "a couple"
	} else if number == 3 { // +1
		return "a few"
	} else { // +1
		return "lots"
	}
} // total complexity = 4

func SumOfPrimes(max int) int { // want "cognitive complexity 7 of func SumOfPrimes is high \\(> 0\\)"
	var total int

OUT:
	for i := 1; i < max; i++ { // +1
		for j := 2; j < i; j++ { // +2 (nesting = 1)
			if i%j == 0 { // +3 (nesting = 2)
				continue OUT // +1
			}
		}
		total += i
		i = 0
	}

	return total
} // Cognitive complexity = 7

func Fibonacci(n int) int { // want "cognitive complexity 3 of func Fibonacci is high \\(> 0\\)"
	if n <= 1 { // +1
		return n
	}

	return Fibonacci(n-1) + Fibonacci(n-2) // +1 and +1
} // Cognitive complexity = 3

func FactRec(n int) int { // want "cognitive complexity 3 of func FactRec is high \\(> 0\\)"
	if n <= 1 { // +1
		return 1
	} else { // +1
		return n * FactRec(n-1) // +1
	}
} // total complexity = 3

func FactRec_Simplified(n int) int { // want "cognitive complexity 2 of func FactRec_Simplified is high \\(> 0\\)"
	if n <= 1 { // +1
		return 1
	}

	return n * FactRec_Simplified(n-1) // +1
} // total complexity = 2

func FactLoop(n int) int { // want "cognitive complexity 1 of func FactLoop is high \\(> 0\\)"
	total := 1
	for n > 0 { // +1
		total *= n
		n--
	}
	return total
} // total complexity = 1

func DumpVal(w io.Writer, i interface{}) error { // want "cognitive complexity 1 of func DumpVal is high \\(> 0\\)"
	switch v := i.(type) { // +1
	case int:
		fmt.Fprint(w, "int ", v)
	case string:
		fmt.Fprint(w, "string", v)
	default:
		return errors.New("unrecognized type")
	}

	return nil
} // total complexity = 1

func ForRange(a []int) int { // want "cognitive complexity 3 of func ForRange is high \\(> 0\\)"
	var sum int
	for _, v := range a { // +1
		sum += v
		if v%2 == 0 { // + 2 (nesting = 1)
			sum += 1
		}
	}

	return sum
} // total complexity = 3

func MyFunc(a bool) { // want "cognitive complexity 6 of func MyFunc is high \\(> 0\\)"
	if a { // +1
		for i := 0; i < 10; i++ { // +2 (nesting = 1)
			n := 0
			for n < 10 { // +3 (nesting = 2)
				n++
			}
		}
	}
} // total complexity = 6

func MyFunc2(a bool) { // want "cognitive complexity 2 of func MyFunc2 is high \\(> 0\\)"
	x := func() { // +0 (but nesting level is now 1)
		if a { // +2 (nesting = 1)
			fmt.Fprintln(io.Discard, "true")
		}
	}

	x()
} // total complexity = 2

//gocognit:ignore
func IgnoreMe(name string) bool {
	if name == "me" { // +1
		return true
	}

	return false
} // total complexity = 1
