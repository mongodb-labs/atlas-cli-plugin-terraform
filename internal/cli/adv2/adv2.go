package adv2

import (
	"errors"

	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "advancedClusterV1ToV2",
		Short:   "Convert advanced_cluster v1 to v2",
		Long:    "Convert a Terraform configuration from mongodbatlas_advanced_cluster schema v1 to v2",
		Aliases: []string{"adv2"},
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("TODO: not implemented yet, will be implemented in the future")
		},
	}
	return cmd
}
