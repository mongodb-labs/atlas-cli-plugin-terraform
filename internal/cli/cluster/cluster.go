package cluster

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster_to_advanced",
		Short:   "Upgrade cluster to advanced_cluster",
		Long:    "WIP - Long description for upgrade cluster to advanced_cluster",
		Aliases: []string{"clu2adv"},
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("WIP - Upgrade cluster to advanced_cluster")
		},
	}
	return cmd
}
