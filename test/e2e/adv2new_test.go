package e2e_test

import (
	"os"
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/test/e2e"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdv2NewParams(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	var (
		prefix         = cwd + "/testdata/"
		fileIn         = prefix + "adv2new.in.tf"
		fileOut        = prefix + "adv2new.out.tf"
		fileExpected   = prefix + "adv2new.expected.tf"
		fileUnexisting = prefix + "adv2new.unexisting.tf"
		fs             = afero.NewOsFs()
	)
	tests := map[string]struct {
		expectedErrContains string
		assertFunc          func(t *testing.T)
		args                []string
	}{
		"no params": {
			expectedErrContains: "required flag(s) \"file\", \"output\" not set",
		},
		"no input file": {
			args:                []string{"--output", fileOut},
			expectedErrContains: "required flag(s) \"file\" not set",
		},
		"no output file": {
			args:                []string{"--file", fileIn},
			expectedErrContains: "required flag(s) \"output\" not set",
		},
		"unexisting input file": {
			args:                []string{"--file", fileUnexisting, "--output", fileOut},
			expectedErrContains: "file must exist: " + fileUnexisting,
		},
		"existing output file without replaceOutput flag": {
			args:                []string{"--file", fileIn, "--output", fileExpected},
			expectedErrContains: "file must not exist: " + fileExpected,
		},
		"basic use": {
			args:       []string{"--file", fileIn, "--output", fileOut},
			assertFunc: func(t *testing.T) { t.Helper(); e2e.CompareFiles(t, fs, fileOut, fileExpected) },
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := e2e.RunAdv2New(tc.args...)
			assert.Equal(t, tc.expectedErrContains == "", err == nil)
			if err == nil {
				assert.Empty(t, resp)
				if tc.assertFunc != nil {
					tc.assertFunc(t)
				}
			} else {
				assert.Contains(t, resp, tc.expectedErrContains)
			}
			_ = fs.Remove(fileOut) // Ensure the output file does not exist in case it was generated in some test case
		})
	}
}
