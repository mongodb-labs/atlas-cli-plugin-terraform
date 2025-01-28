package cluster

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	opts := &Opts{fs: afero.NewOsFs()}
	cmd := &cobra.Command{
		Use:     "cluster_to_advanced",
		Short:   "Upgrade cluster to advanced_cluster",
		Long:    "Upgrade Terraform config from mongodbatlas_cluster to mongodbatlas_advanced_cluster",
		Aliases: []string{"clu2adv"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if exists, err := afero.Exists(opts.fs, opts.file); !exists || err != nil {
				return fmt.Errorf("input file not found: %s", opts.file)
			}
			if exists, err := afero.Exists(opts.fs, opts.output); exists || err != nil {
				return fmt.Errorf("output file can't exist: %s", opts.output)
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.Run()
		},
	}
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "input file")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output file")
	_ = cmd.MarkFlagFilename("file")
	_ = cmd.MarkFlagRequired("file")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

type Opts struct {
	fs     afero.Fs
	file   string
	output string
}

func (o *Opts) Run() error {
	content, err := afero.ReadFile(o.fs, o.file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", o.file, err)
	}
	if err := afero.WriteFile(o.fs, o.output, content, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", o.output, err)
	}
	return nil
}
