package gocognit

type Formatter interface {
	Format([]Stat) error
}
