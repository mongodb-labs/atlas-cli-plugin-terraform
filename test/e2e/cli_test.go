package e2e_test

import (
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/test/e2e"
	"github.com/stretchr/testify/require"
)

func TestPlugin(t *testing.T) {
	t.Run("Execute TF command", func(t *testing.T) {
		resp, err := e2e.RunPlugin("tf")
		require.NoError(t, err, resp)
	})
}
