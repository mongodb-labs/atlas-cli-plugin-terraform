package adv2new

import (
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/convert"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/flag"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	o := &cli.BaseOpts{
		Fs:        afero.NewOsFs(),
		Converter: cli.ConvertFunc(convert.AdvancedClusterToNew),
	}
	cmd := &cobra.Command{
		Use:   "advancedClusterToNew",
		Short: "Convert advanced_cluster from provider version 1 to 2",
		Long: "Convert a Terraform configuration from mongodbatlas_advanced_cluster in provider version 1.X.X (SDKv2)" +
			" to version 2.X.X (TPF - Terraform Plugin Framework)",
		Aliases: []string{"adv2new"},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := o.PreRun(); err != nil {
				return err
			}
			return o.Run()
		},
	}
	cmd.Flags().StringVarP(&o.File, flag.File, flag.FileShort, "", "input file")
	_ = cmd.MarkFlagRequired(flag.File)
	cmd.Flags().StringVarP(&o.Output, flag.Output, flag.OutputShort, "", "output file")
	_ = cmd.MarkFlagRequired(flag.Output)
	cmd.Flags().BoolVarP(&o.ReplaceOutput, flag.ReplaceOutput, flag.ReplaceOutputShort, false,
		"replace output file if exists")
	cmd.Flags().BoolVarP(&o.Watch, flag.Watch, flag.WatchShort, false,
		"keeps the plugin running and watches the input file for changes")
	return cmd
}
