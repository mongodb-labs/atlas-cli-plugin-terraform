package cluster

import (
	"fmt"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/file"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	o := &opts{fs: afero.NewOsFs()}
	cmd := &cobra.Command{
		Use:     "cluster_to_advanced",
		Short:   "Upgrade cluster to advanced_cluster",
		Long:    "Upgrade Terraform config from mongodbatlas_cluster to mongodbatlas_advanced_cluster",
		Aliases: []string{"clu2adv"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return o.PreRun()
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return o.Run()
		},
	}
	cmd.Flags().StringVarP(&o.file, "file", "f", "", "input file")
	_ = cmd.MarkFlagRequired("file")
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "output file")
	_ = cmd.MarkFlagRequired("output")
	cmd.Flags().BoolVarP(&o.overwriteOutput, "overwriteOutput", "w", false, "overwrite output file if exists")
	return cmd
}

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
	content, err := afero.ReadFile(o.fs, o.file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", o.file, err)
	}
	if err := afero.WriteFile(o.fs, o.output, content, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", o.output, err)
	}
	return nil
}
