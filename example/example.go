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
	if a != 0 && a != 10 { // +2
		return a + 1
	}
	return a
}

func funcIfElse2(a int) int {
	if a > 0 { // +1
		a++
	} else if a < 0 { // +2
		a--
	}
	return a
}

func funcIfElse3(a int) int {
	if a > 10 { // +1
		a++
	} else if a < 0 || a > 1 { // +2
		a--
	}
	return a
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

func funcSwitch5(a int) int {
	switch {
	case a == 0: // +1
		return a
	case a > 10000 || a < 100 || a == 200: // +3
		return a * a
	case a < -10: // +1
		return -a * a
	}
	return a + 100
}

func funcSwitchType1(a interface{}) int {
	switch v := a.(type) {
	case int:
		return v
	case int64:
		return -1
	default:
		return 0
	}
}
