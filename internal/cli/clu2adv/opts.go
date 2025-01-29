package clu2adv

import (
	"fmt"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/file"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/spf13/afero"
)

type opts struct {
	fs              afero.Fs
	file            string
	output          string
	overwriteOutput bool
}

func (o *opts) PreRun() error {
	if err := file.MustExist(o.fs, o.file); err != nil {
		return err
	}
	if !o.overwriteOutput {
		return file.MustNotExist(o.fs, o.output)
	}
	return nil
}

func (o *opts) Run() error {
	inConfig, err := afero.ReadFile(o.fs, o.file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", o.file, err)
	}
	outConfig, err := hcl.ClusterToAdvancedCluster(inConfig)
	if err != nil {
		return err
	}
	if err := afero.WriteFile(o.fs, o.output, outConfig, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", o.output, err)
	}
	return nil
}
