/*
 * Copyright (C) distroy
 */

package example

func funcIf1(a int) int {
	if a != 0 { // +1
		return a + 1
	}
	return a
}

func funcIf2(a int) int {
	if a != 0 && (a != 10) { // +2
		return a + 1
	}
	return a
}

func funcIf4(a, b, c, d bool) int {
	if a && (b && c || d) { // +4
		return 10
	}
	return 0
}

func funcIfElse2(a int) int {
	if a > 0 { // +1
		a++
	} else if a < 0 { // +2
		a--
	}
	return a
}

func funcIfElse4(a int) int {
	if a > 10 { // +1
		a++
	} else if a < 0 || a > 1 { // +2
		a--
	} else { // +1
		a += 100
	}
	return a
}

func funcIfElse6(a, b, c int) int {
	if a > 10 { // +1
		return a

	} else if a > 1 { // +1
		if b < 10000 { // +2
			return b
		} else if b < 10 { // +1
			return a + b
		} else { // +1
			c = c * a
		}
	}
	return b + c
}

func funcFor1(a int) int {
	s := 0
	for i := 0; i < a; i++ {
		s += i
	}
	return s
}

func funcFor3(a int) int {
	s := 0
	for i := 0; i < a; i++ {
		if i != 10 {
			s += i
		}
	}
	return s
}

func funcBreak3(a int) int {
	sum := 0
	for i := 0; i < a; i++ { // +1
		if i > 100 { // +2
			break
		}
		sum += i
	}
	return sum
}

func funcBreak7(a int) int {
	sum := 0
out:
	for i := 0; i < a; i++ { // +1
		for j := 0; j < a; j++ { // +2
			sum += i * j
			if j > 100 { // +3
				break out // +1
			}
		}
	}
	return sum
}

func funcFunc4(a int) func(b int) int {
	return func(b int) int {
		if a < 0 { // +2
			return -a
		}
		if a < 10 { // +2
			return a + b
		}
		return a * b
	}
}

func funcFunc9(a, b, c, d int) func(x int) int {
	switch { // +1
	default:
		return func(x int) int { return x }

	case a > 10000: // +1
		return func(x int) int {
			if x < 10000 { // +3
				return c + x
			}
			return a + x
		}

	case a > 100: // +1
		return func(x int) int {
			if x > 100 { // +3
				return d + x
			}
			return b + x
		}
	}
}

func funcSwitch1(a int) int {
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

func funcSwitch3(a int) int {
	switch b := a + 1; b { // +1
	case 0:
		return b
	case 1:
		return -b
	case -1, 2:
		return a - 1
	case 3:
		if b < 3 { // +2
			return a * b
		}
		return a + b
	}
	return 0
}

func funcSwitch4(a int) int {
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

func funcSwitchType1(a interface{}) int {
	switch v := a.(type) { // +1
	case int:
		return v
	case int64:
		return -1
	default:
		return 0
	}
}

func funcIfSwith9(a, b, c, d int) int {
	if a == 0 { // +1
		switch b { // +2
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
		case c > 10000: // +2
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

func funcSelect1() int {
	a := make(chan int)
	b := make(chan int)
	select { // +1
	case v := <-a:
		return v
	case v := <-b:
		return v
	}
}
