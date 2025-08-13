package adv2new

import (
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/convert"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	o := &cli.BaseOpts{
		Fs:      afero.NewOsFs(),
		Convert: convert.AdvancedClusterToNew,
	}
	cmd := &cobra.Command{
		Use:   "advancedClusterToNew",
		Short: "Convert advanced_cluster from provider version 1 to 2",
		Long: "Convert a Terraform configuration from mongodbatlas_advanced_cluster in provider version 1.X.X (SDKv2)" +
			" to version 2.X.X (TPF - Terraform Plugin Framework)",
		Aliases: []string{"adv2new"},
		RunE:    o.RunE,
	}
	cli.SetupCommonFlags(cmd, o)
	return cmd
}
