package main

import (
	"fmt"
	"os"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli/adv2v2"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli/clu2adv"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli/moduleimport"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/flags"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/logger"
	"github.com/spf13/cobra"
)

func main() {
	var (
		debugLevel bool
	)
	terraformCmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Utilities for Terraform's MongoDB Atlas Provider",
		Aliases: []string{"tf"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logger.SetWriter(cmd.ErrOrStderr())
			if debugLevel {
				logger.SetLevel(logger.DebugLevel)
			}

			return nil
		},
	}
	terraformCmd.AddCommand(clu2adv.Builder())
	terraformCmd.AddCommand(adv2v2.Builder())
	terraformCmd.AddCommand(moduleimport.Builder())

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

	rootCmd.PersistentFlags().BoolVarP(&debugLevel, flags.Debug, flags.DebugShort, false, "Debug log level.")
	_ = rootCmd.PersistentFlags().MarkHidden(flags.Debug)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
