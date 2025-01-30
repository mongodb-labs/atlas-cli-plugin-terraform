package hcl_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/sebdah/goldie/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterToAdvancedCluster(t *testing.T) {
	const (
		root      = "testdata/clu2adv"
		inSuffix  = ".in.tf"
		outSuffix = ".out.tf"
	)
	g := goldie.New(t,
		goldie.WithFixtureDir(root),
		goldie.WithNameSuffix(outSuffix))
	fs := afero.NewOsFs()
	pattern := filepath.Join(root, "*"+inSuffix)
	inputFiles, err := afero.Glob(fs, pattern)
	require.NoError(t, err)
	assert.NotEmpty(t, inputFiles)
	for _, inputFile := range inputFiles {
		testName := strings.TrimSuffix(filepath.Base(inputFile), inSuffix)
		inConfig, err := afero.ReadFile(fs, inputFile)
		require.NoError(t, err)
		outConfig, err := hcl.ClusterToAdvancedCluster(inConfig)
		require.NoError(t, err)
		g.Assert(t, testName, outConfig)
	}
}
