package moduleimport

import (
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/log"
	"github.com/spf13/cobra"
)

type ModuleImportOpts struct {
	example string
}

// atlas terraform module-import -f ./local-testing/converter/input.tf -o ./local-testing/converter/out.tf
func Builder() *cobra.Command {
	opts := &ModuleImportOpts{}
	cmd := &cobra.Command{
		Use:     "module-import",
		Short:   "Generate Terraform module configurations",
		Long:    "Generate Terraform module configurations to import existing infrastructure",
		PreRunE: opts.PreRun,
		RunE:    opts.Run,
	}

	cmd.Flags().StringVarP(&opts.example, "example", "e", "", "example flag")
	return cmd
}

func (opts *ModuleImportOpts) PreRun(cmd *cobra.Command, args []string) error {
	log.Debug("[module-import] PreRunE\n")
	return nil
}

func (opts *ModuleImportOpts) Run(cmd *cobra.Command, args []string) error {
	log.Debugf("[module-import] RunE - example: %s\n", opts.example)
	return nil
}
