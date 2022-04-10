/*
 * Copyright (C) distroy
 */

package example

import (
	"log"
)

// doc: https://sonarsource.com/docs/CognitiveComplexity.pdf

func funcIf_1(a int) int {
	if a != 0 { // +1
		return a + 1
	}
	return a
}

func funcIf_2(a int) int {
	if a != 0 && (a != 10) { // +2
		return a + 1
	}
	return a
}

func funcIf_4(a, b, c, d bool) int {
	if a && (b && c || d) { // +4
		return 10
	}
	return 0
}

func funcIfElse_2(a int) int {
	if a > 0 { // +1
		a++
	} else if a < 0 { // +1
		a--
	}
	return a
}

func funcIfElse_4(a int) int {
	if a > 10 { // +1
		a++
	} else if a < 0 || a > 1 { // +2
		a--
	} else { // +1
		a += 100
	}
	return a
}

func funcIfElse_6(a, b, c int) int {
	if a > 10 { // +1
		return a

	} else if a > 1 { // +1
		if b < 10000 { // +2 (nesting +1)
			return b
		} else if b < 10 { // +1
			return a + b
		} else { // +1
			c = c * a
		}
	}
	return b + c
}

func funcIfParen_5(a, b, c, d, e int) int {
	if (a > 10 || a < -100) && (!(b < 10000 && b > 10 && d > 1000) || c > 100) && e > 10 { // +5
		return a
	}
	return b
}

func funcLogicalExpr_1(a, b, c, d, e int) bool {
	return a > 10 || !(b > 1000) || c < 100 // +1
}

func funcLogicalExpr_2(a, b, c, d, e int) bool {
	return a > 10 || b > 1000 && c < 100 // +2
}

func funcLogicalExpr_3(a, b, c, d, e int) bool {
	return a > 10 || b > 1000 && c < 100 || d > -10 // +3
}

func funcLogicalExpr_4(a, b, c, d, e int) bool {
	return (a > 10 || a < -100) && ((b < 10000 && b > 10 && d > 1000) || c > 100) && e > 10 // +4
}

func funcLogicalExpr_5(a, b, c, d, e int) bool {
	return (a > 10 || a < -100) && ((b < 10000 && (b > 10 && d > 1000)) || c > 100) && e > 10 // +5
}

func funcFor_1(a int) int {
	s := 0
	for i := 0; i < a; i++ { // +1
		s += i
	}
	return s
}

func funcFor_3(a int) int {
	s := 0
	for i := 0; i < a; i++ { // +1
		if i != 10 { // +2 (nesting +1)
			s += i
		}
	}
	return s
}

func funcBreak_3(a int) int {
	sum := 0
	for i := 0; i < a; i++ { // +1
		if i > 100 { // +2 (nesting +1)
			break
		}
		sum += i
	}
	return sum
}

func funcBreakLabel_7(a int) int {
	sum := 0
out:
	for i := 0; i < a; i++ { // +1
		for j := 0; j < a; j++ { // +2 (nesting +1)
			sum += i * j
			if j > 100 { // +3 (nesting +2)
				break out // +1
			}
		}
	}
	return sum
}

func funcFunc_4(a int) func(b int) int {
	return func(b int) int {
		if a < 0 { // +2 (nesting +1)
			return -a
		}
		if a < 10 { // +2 (nesting +1)
			return a + b
		}
		return a * b
	}
}

func funcRecursion_3(n int) int {
	if n <= 1 { // +1
		return 1
	} else { // +1
		return n * funcRecursion_3(n-1) // +1
	}
}

func funcFunc_9(a, b, c, d int) func(x int) int {
	switch { // +1
	default:
		return func(x int) int { return x }

	case a > 10000: // +1
		return func(x int) int {
			if x < 10000 { // +3 (nesting +2)
				return c + x
			}
			return a + x
		}

	case a > 100: // +1
		return func(x int) int {
			if x > 100 { // +3 (nesting +2)
				return d + x
			}
			return b + x
		}
	}
}

func funcSwitch_1(a int) int {
	switch a { // +1
	case 0:
		return a
	case 1:
		return -a
	case -1, 2:
		return a - 1
	case 3:
		return a
	}
	return -1
}

func funcSwitch_3(a int) int {
	switch b := a + 1; b { // +1
	case 0:
		return b
	case 1:
		return -b
	case -1, 2:
		return a - 1
	case 3:
		if b < 3 { // +2 (nesting +1)
			return a * b
		}
		return a + b
	}
	return 0
}

func funcSwitch_4(a int) int {
	switch {
	case a == 0: // +1
		return a
	case a > 10000 || a < 100 || a == 200: // +2
		return a * a
	case a < -10: // +1
		return -a * a
	}
	return a + 100
}

func funcSwitchType_1(a interface{}) int {
	switch v := a.(type) { // +1
	case int:
		return v
	case int64:
		return -1
	default:
		return 0
	}
}

func funcIfSwith_9(a, b, c, d int) int {
	if a == 0 { // +1
		switch b { // +2 (nesting +1)
		case 1:
			return 2
		case 2:
			return -1
		case -1:
			return 1
		default:
			return b
		}
	} else if a > 1000 { // +1
		switch {
		case c > 10000: // +2 (nesting +1)
			return c
		case c > 100: // +1
			return b
		case c > 10: // +1
			return d
		default: // +1
			return a
		}
	}

	switch {
	}

	return a + b
}

func funcSelect_1() int {
	a := make(chan int)
	b := make(chan int)
	select { // +1
	case v := <-a:
		return v
	case v := <-b:
		return v
	}
}

func funcDefer_2(a, b, c int) int {
	defer func() {
		if err := recover(); err != nil { // +2 (nesting +1)
			log.Print("recover: ", err)
		}
	}()
	return (a + b) % c
}
