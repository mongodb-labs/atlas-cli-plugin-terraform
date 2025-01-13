package hello

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "The Hello World command",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("Hello World, Terraform! This command will be eventually deleted.")
		},
	}
}
