package cli

import (
	"errors"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/file"
	"github.com/spf13/afero"
)

// Converter defines the interface for different conversion functions.
type Converter interface {
	Convert(config []byte) ([]byte, error)
}

// ConvertFunc is a function type that implements the Converter interface.
type ConvertFunc func(config []byte) ([]byte, error)

func (f ConvertFunc) Convert(config []byte) ([]byte, error) {
	return f(config)
}

// BaseOpts contains common functionality for CLI commands that convert files.
type BaseOpts struct {
	Fs            afero.Fs
	Converter     Converter
	File          string
	Output        string
	ReplaceOutput bool
	Watch         bool
}

// PreRun validates the input and output files before running the command.
func (o *BaseOpts) PreRun() error {
	if err := file.MustExist(o.Fs, o.File); err != nil {
		return err
	}
	if !o.ReplaceOutput {
		return file.MustNotExist(o.Fs, o.Output)
	}
	return nil
}

// Run executes the conversion and optionally watches for file changes.
func (o *BaseOpts) Run() error {
	if err := o.generateFile(false); err != nil {
		return err
	}
	if o.Watch {
		return o.watchFile()
	}
	return nil
}

// generateFile reads the input file, converts it, and writes the output.
func (o *BaseOpts) generateFile(allowParseErrors bool) error {
	inConfig, err := afero.ReadFile(o.Fs, o.File)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", o.File, err)
	}

	outConfig, err := o.Converter.Convert(inConfig)
	if err != nil {
		if allowParseErrors {
			outConfig = []byte("# CONVERT ERROR: " + err.Error() + "\n\n")
			outConfig = append(outConfig, inConfig...)
		} else {
			return err
		}
	}

	if err := afero.WriteFile(o.Fs, o.Output, outConfig, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", o.Output, err)
	}
	return nil
}

// watchFile watches the input file for changes and regenerates the output.
func (o *BaseOpts) watchFile() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(o.File); err != nil {
		return err
	}

	for {
		if err := o.waitForFileEvent(watcher); err != nil {
			return err
		}
	}
}

// waitForFileEvent waits for file system events and regenerates the output file.
func (o *BaseOpts) waitForFileEvent(watcher *fsnotify.Watcher) error {
	watcherError := errors.New("watcher has been closed")
	select {
	case event, ok := <-watcher.Events:
		if !ok {
			return watcherError
		}
		if event.Has(fsnotify.Write) {
			if err := o.generateFile(true); err != nil {
				return err
			}
		}
	case err, ok := <-watcher.Errors:
		if !ok {
			return watcherError
		}
		return err
	}
	return nil
}
