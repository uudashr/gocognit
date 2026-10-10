package testdata

import (
	"errors"
)

func SimpleErrCheck(err error) error {
	if err != nil {
		return err
	}

	return nil
}

func ErrCheckWithInit() error {
	if err := doStep(); err != nil {
		return err
	}

	return nil
}

func ComplexWithErrCheck(err error, n int) string { // want "cognitive complexity 1 of func ComplexWithErrCheck is high \\(> 0\\)"
	if err != nil {
		return "error"
	}

	if n > 10 { // +1
		return "big"
	}

	return "small"
}

func doStep() error {
	return errors.New("fail")
}
