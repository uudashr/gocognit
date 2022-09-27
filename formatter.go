package gocognit

type Formatter interface {
	Write([]Stat) error
}
