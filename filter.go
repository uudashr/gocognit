package gocognit

type FilterFunc func(stat Stat, index int) bool

type Filter struct {
	filterFuncs []FilterFunc
}

func (f *Filter) Apply(original []Stat) []Stat {
	filtered := make([]Stat, len(original))

	var numEntries int
	// loop all stats
	for i, stat := range original {
		keep := true
		numEntries = i

		// loop all filters
		for _, filter := range f.filterFuncs {
			keep = filter(stat, i)

			if !keep {
				break
			}
		}

		if keep {
			filtered[i] = stat
		}
	}

	return filtered[:numEntries-1]
}

func (f *Filter) AddFilter(filterFunc FilterFunc) {
	f.filterFuncs = append(f.filterFuncs, filterFunc)
}
