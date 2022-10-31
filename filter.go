package gocognit

type FilterFunc func(stat Stat, index int) bool

func NewTopFilter(top int) FilterFunc {
	return func(_ Stat, i int) bool {
		return i < top
	}
}

func NewComplexityFilter(over int) FilterFunc {
	return func(stat Stat, _ int) bool {
		return stat.Complexity > over
	}
}

type Filter struct {
	filterFuncs []FilterFunc
}

func (f *Filter) Apply(original []Stat) []Stat {
	filtered := make([]Stat, 0, len(original))

	// loop all stats
	for i, stat := range original {
		keep := true

		// loop all filters
		for _, filter := range f.filterFuncs {
			keep = filter(stat, i)

			if !keep {
				break
			}
		}

		if keep {
			filtered = append(filtered, stat)
		}
	}

	return filtered
}

func (f *Filter) AddFilter(filterFunc FilterFunc) {
	f.filterFuncs = append(f.filterFuncs, filterFunc)
}
