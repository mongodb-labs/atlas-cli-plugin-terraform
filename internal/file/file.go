package file

import (
	"fmt"

	"github.com/spf13/afero"
)

func Exists(fs afero.Fs, filename string) (exists bool, err error) {
	exists, err = afero.Exists(fs, filename)
	if err != nil {
		return false, newError(err, filename)
	}
	return
}

func MustExist(fs afero.Fs, filename string) error {
	exists, err := Exists(fs, filename)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("file must exist: %s", filename)
	}
	return nil
}

func MustNotExist(fs afero.Fs, filename string) error {
	exists, err := Exists(fs, filename)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("file must not exist: %s", filename)
	}
	return nil
}

func newError(err error, filename string) error {
	return fmt.Errorf("error in file %s: %w", filename, err)
}
