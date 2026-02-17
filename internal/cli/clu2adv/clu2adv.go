package clu2adv

import (
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/convert"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/flags"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	o := &struct {
		cli.BaseOpts
		includeMoved bool
	}{
		BaseOpts: cli.BaseOpts{
			Fs: afero.NewOsFs(),
		},
	}
	o.Convert = func(config []byte) ([]byte, error) {
		return convert.ClusterToAdvancedCluster(config, o.includeMoved)
	}
	cmd := &cobra.Command{
		Use:   "clusterToAdvancedCluster",
		Short: "Convert cluster to advanced_cluster preview provider 2.0.0",
		Long: "Convert a Terraform configuration from mongodbatlas_cluster to " +
			"mongodbatlas_advanced_cluster preview provider 2.0.0",
		Aliases: []string{"clu2adv"},
		RunE:    o.RunE,
	}
	cli.SetupCommonFlags(cmd, &o.BaseOpts)
	cmd.Flags().BoolVarP(&o.includeMoved, flags.IncludeMoved, flags.IncludeMovedShort, false,
		"include moved blocks in the output file")
	return cmd
}
