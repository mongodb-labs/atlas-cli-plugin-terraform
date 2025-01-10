package main

import (
	"fmt"
	"os"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/cli/hello"
	"github.com/spf13/cobra"
)

func main() {
	terraformCmd := &cobra.Command{
		Use:   "terraform",
		Short: "Root command of the Atlas CLI plugin for MongoDB Atlas Provider",
	}

	terraformCmd.AddCommand(
		hello.Builder(),
	)

	completionOption := &cobra.CompletionOptions{
		DisableDefaultCmd:   true,
		DisableNoDescFlag:   true,
		DisableDescriptions: true,
		HiddenDefaultCmd:    true,
	}
	rootCmd := &cobra.Command{
		Aliases:            []string{"tf"},
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
