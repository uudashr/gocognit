package gocognit

import (
	"encoding/json"
	"io"
)

type JsonFormatter struct {
	writer          io.Writer
	withIndentation bool
}

func NewJsonFormatter(writer io.Writer, withIndentation bool) JsonFormatter {
	return JsonFormatter{writer, withIndentation}
}

func (t JsonFormatter) Format(stats []Stat) error {
	encoder := json.NewEncoder(t.writer)

	if t.withIndentation {
		encoder.SetIndent("", "\t")
	}

	return encoder.Encode(stats)
}
