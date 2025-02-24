package e2e_test

import (
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/test/e2e"
	"github.com/stretchr/testify/require"
)

func TestRunTF(t *testing.T) {
	resp, err := e2e.RunTF()
	require.NoError(t, err, resp)
}
