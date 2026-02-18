package cli

import (
	"errors"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/file"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/flags"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type ConvertFn func(config []byte) ([]byte, error)

// BaseOpts contains common functionality for CLI commands that convert files.
type BaseOpts struct {
	Fs            afero.Fs
	Convert       ConvertFn
	File          string
	Output        string
	ReplaceOutput bool
	Watch         bool
}

// RunE is the entry point for the command.
func (o *BaseOpts) RunE(cmd *cobra.Command, args []string) error {
	if err := o.preRun(); err != nil {
		return err
	}
	return o.run()
}

// preRun validates the input and output files before running the command.
func (o *BaseOpts) preRun() error {
	if err := file.MustExist(o.Fs, o.File); err != nil {
		return err
	}
	if !o.ReplaceOutput {
		return file.MustNotExist(o.Fs, o.Output)
	}
	return nil
}

// run executes the conversion and optionally watches for file changes.
func (o *BaseOpts) run() error {
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

	outConfig, err := o.Convert(inConfig)
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

// SetupCommonFlags sets up the common flags used by all commands.
func SetupCommonFlags(cmd *cobra.Command, opts *BaseOpts) {
	cmd.Flags().StringVarP(&opts.File, flags.File, flags.FileShort, "", "input file")
	_ = cmd.MarkFlagRequired(flags.File)
	cmd.Flags().StringVarP(&opts.Output, flags.Output, flags.OutputShort, "", "output file")
	_ = cmd.MarkFlagRequired(flags.Output)
	cmd.Flags().BoolVarP(&opts.ReplaceOutput, flags.ReplaceOutput, flags.ReplaceOutputShort, false,
		"replace output file if exists")
	cmd.Flags().BoolVarP(&opts.Watch, flags.Watch, flags.WatchShort, false,
		"keeps the plugin running and watches the input file for changes")
}
