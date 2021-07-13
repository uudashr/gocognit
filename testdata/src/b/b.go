package testdata

import (
	"errors"
	"fmt"
	"io"
)

func GetWords_IfElse(number int) string { // want "cognitive complexity 4 of func GetWords_IfElse is high \\(> 3\\)"
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

func GetWords_SwitchCase(number int) string {
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

func SumOfPrimes(max int) int { // want "cognitive complexity 7 of func SumOfPrimes is high \\(> 3\\)"
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

func Fact(n int) int {
	if n <= 1 { // +1
		return 1
	} else { // +1
		return n * Fact(n-1) // +1
	}
} // total complexity = 3

func DumpVal(w io.Writer, i interface{}) error {
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
