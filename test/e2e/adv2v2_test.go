package e2e_test

import (
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/test/e2e"
)

func TestAdvancedClusterToV2(t *testing.T) {
	e2e.RunTests(t, "adv2v2", nil)
}
