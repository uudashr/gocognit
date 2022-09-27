package gocognit

import (
	"fmt"
	"io"
)

type TextFormatter struct {
	writer io.Writer
}

func NewTextFormatter(writer io.Writer) TextFormatter {
	return TextFormatter{writer: writer}
}

func (t TextFormatter) Write(stats []Stat) error {

	for _, stat := range stats {
		_, err := fmt.Fprintln(t.writer, stat)

		if err != nil {
			return err
		}
	}

	return nil
}
