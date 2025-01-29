package clu2adv

import (
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
