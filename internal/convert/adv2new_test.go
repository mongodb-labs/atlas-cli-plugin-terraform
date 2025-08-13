package convert_test

import (
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/convert"
)

func TestAdvancedClusterToNew(t *testing.T) {
	runConvertTests(t, "adv2new", func(testName string, inConfig []byte) ([]byte, error) {
		return convert.AdvancedClusterToNew(inConfig)
	})
}
