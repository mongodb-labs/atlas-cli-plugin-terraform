package clu2adv

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	o := &opts{fs: afero.NewOsFs()}
	cmd := &cobra.Command{
		Use:     "clusterToAdvancedCluster",
		Short:   "Convert cluster to advanced_cluster v2",
		Long:    "Convert a Terraform configuration from mongodbatlas_cluster to mongodbatlas_advanced_cluster schema v2",
		Aliases: []string{"clu2adv"},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := o.PreRun(); err != nil {
				return err
			}
			return o.Run()
		},
	}
	cmd.Flags().StringVarP(&o.file, "file", "f", "", "input file")
	_ = cmd.MarkFlagRequired("file")
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "output file")
	_ = cmd.MarkFlagRequired("output")
	cmd.Flags().BoolVarP(&o.replaceOutput, "replaceOutput", "r", false, "replace output file if exists")
	cmd.Flags().BoolVarP(&o.watch, "watch", "w", false, "keeps the command running and watches the input file for changes")
	return cmd
}
