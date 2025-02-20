package clu2adv

import (
	"errors"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/convert"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/file"
	"github.com/spf13/afero"
)

type opts struct {
	fs            afero.Fs
	file          string
	output        string
	replaceOutput bool
	watch         bool
}

func (o *opts) PreRun() error {
	if err := file.MustExist(o.fs, o.file); err != nil {
		return err
	}
	if !o.replaceOutput {
		return file.MustNotExist(o.fs, o.output)
	}
	return nil
}

func (o *opts) Run() error {
	if err := o.generateFile(false); err != nil {
		return err
	}
	if o.watch {
		return o.watchFile()
	}
	return nil
}

func (o *opts) generateFile(allowParseErrors bool) error {
	inConfig, err := afero.ReadFile(o.fs, o.file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", o.file, err)
	}
	outConfig, err := convert.ClusterToAdvancedCluster(inConfig)
	if err != nil {
		if allowParseErrors {
			outConfig = []byte("# CONVERT ERROR: " + err.Error() + "\n\n")
			outConfig = append(outConfig, inConfig...)
		} else {
			return err
		}
	}
	if err := afero.WriteFile(o.fs, o.output, outConfig, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", o.output, err)
	}
	return nil
}

func (o *opts) watchFile() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	defer watcher.Close()
	err = watcher.Add(o.file)
	if err != nil {
		return err
	}
	watcherError := errors.New("watcher has been closed")
	for {
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
	}
}
