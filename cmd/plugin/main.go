package main

import (
	"fmt"
	"os"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli/cluster"
	"github.com/spf13/cobra"
)

func main() {
	terraformCmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Utilities for Terraform's MongoDB Atlas Provider",
		Aliases: []string{"tf"},
	}

	terraformCmd.AddCommand(
		cluster.Builder(),
	)

	completionOption := &cobra.CompletionOptions{
		DisableDefaultCmd:   true,
		DisableNoDescFlag:   true,
		DisableDescriptions: true,
		HiddenDefaultCmd:    true,
	}
	rootCmd := &cobra.Command{
		DisableFlagParsing: true,
		DisableAutoGenTag:  true,
		DisableSuggestions: true,
		CompletionOptions:  *completionOption,
	}
	rootCmd.AddCommand(terraformCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
