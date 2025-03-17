package clu2adv

import (
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/flag"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	o := &opts{fs: afero.NewOsFs()}
	cmd := &cobra.Command{
		Use:     "clusterToAdvancedCluster",
		Short:   "Convert cluster to advanced_cluster preview provider 2.0.0",
		Long:    "Convert a Terraform configuration from mongodbatlas_cluster to mongodbatlas_advanced_cluster preview provider 2.0.0",
		Aliases: []string{"clu2adv"},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := o.PreRun(); err != nil {
				return err
			}
			return o.Run()
		},
	}
	cmd.Flags().StringVarP(&o.file, flag.File, flag.FileShort, "", "input file")
	_ = cmd.MarkFlagRequired(flag.File)
	cmd.Flags().StringVarP(&o.output, flag.Output, flag.OutputShort, "", "output file")
	_ = cmd.MarkFlagRequired(flag.Output)
	cmd.Flags().BoolVarP(&o.replaceOutput, flag.ReplaceOutput, flag.ReplaceOutputShort, false, "replace output file if exists")
	cmd.Flags().BoolVarP(&o.watch, flag.Watch, flag.WatchShort, false, "keeps the plugin running and watches the input file for changes")
	cmd.Flags().BoolVarP(&o.includeMoved, flag.IncludeMoved, flag.IncludeMovedShort, false, "include moved blocks in the output file")
	return cmd
}
