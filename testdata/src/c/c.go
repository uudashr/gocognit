package testdata

type Node[T any] struct {
}

func (n *Node[T]) String() string { // want "cognitive complexity 1 of func \\(\\*Node\\)\\.String is high \\(> 0\\)"
	if n != nil { // +1
		return "Node"
	}

	return ""
} // total complexity = 1

type Pair[K any, V any] struct {
	Key   K
	Value V
}

func (p *Pair[K, V]) String() string { // want "cognitive complexity 1 of func \\(\\*Pair\\)\\.String is high \\(> 0\\)"
	if p != nil { // +1
		return "Pair"
	}

	return ""
} // total complexity = 1

type Triple[K any, V any, T any] struct {
}

func (t *Triple[K, V, T]) String() string { // want "cognitive complexity 1 of func \\(\\*Triple\\)\\.String is high \\(> 0\\)"
	if t != nil { // +1 `
		return "Triple"
	}

	return ""
} // total complexity = 1

type Number interface {
	int64 | float64
}

func SumNumbers[K comparable, V Number](m map[K]V) V { // want "cognitive complexity 1 of func SumNumbers is high \\(> 0\\)"
	var s V
	for _, v := range m {
		s += v
	}
	return s
} // total complexity = 1
