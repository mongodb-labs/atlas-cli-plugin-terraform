package modulegen_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/modulegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/atlas-sdk/v20250312014/auth/clientcredentials"
	"golang.org/x/tools/txtar"
)

// This is an early WIP example of how to test the import tool showcasing how we can run mock and e2e tests.

var (
	update = flag.Bool(
		"update", false,
		"update expected.txtar files in mock tests instead of checking against existing ones",
	)
	record = flag.Bool(
		"record", false,
		"save fetched resources `ResourceStore` into a json under `recordings/` during e2e tests",
	)
)

func atlasBaseURL() string {
	if url := os.Getenv("MONGODB_ATLAS_BASE_URL"); url != "" {
		return url
	}
	return modulegen.CloudDevServiceURL
}

// TestMock runs a test for each sub-dir under `mocktestdata/`.
// Can run specific tests by setting the MOCK_TEST_REGEX env var (e.g. MOCK_TEST_REGEX="^project").
// Each mock dir contains: `input.tfvars`, `resource_store.json` & `expected.txtar`.
// If the -update flag is set to true, the `expected.txtar` is created/updated with the output of modulegen.Run.
func TestMock(t *testing.T) {
	var filter *regexp.Regexp
	if pattern := os.Getenv("MOCK_TEST_REGEX"); pattern != "" {
		var err error
		filter, err = regexp.Compile(pattern)
		require.NoError(t, err, "invalid MOCK_TEST_REGEX expression")
	}

	entries, err := os.ReadDir("mocktestdata")
	require.NoError(t, err)
	for _, entry := range entries {
		require.True(t, entry.IsDir())
		if filter != nil && !filter.MatchString(entry.Name()) {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			runMockTest(t, filepath.Join("mocktestdata", entry.Name()))
		})
	}
}

// runMockTest runs modulegen.Run with a mockClient and compares the output against the expected.txtar in the directory.
func runMockTest(t *testing.T, dir string) {
	t.Helper()
	inputFile := filepath.Join(dir, "input.tfvars")
	resourceStoreFile := filepath.Join(dir, "resource_store.json")
	expectedFile := filepath.Join(dir, "expected.txtar")

	outDir := t.TempDir()

	err := modulegen.Run(
		t.Context(),
		&modulegen.GenArgs{
			InputPath:    inputFile,
			OutputPath:   outDir,
			AtlasBaseURL: modulegen.CloudServiceURL,
		},
		&mockClient{path: resourceStoreFile},
	)
	require.NoError(t, err)

	if *update {
		entries, err := os.ReadDir(outDir) //nolint:govet // err shadowing
		require.NoError(t, err)
		var archive txtar.Archive
		for _, entry := range entries {
			data, err := os.ReadFile(filepath.Join(outDir, entry.Name()))
			require.NoError(t, err)
			archive.Files = append(archive.Files, txtar.File{Name: entry.Name(), Data: data})
		}
		require.NoError(t, os.WriteFile(expectedFile, txtar.Format(&archive), 0o600))
		fmt.Printf("[runMockTest] updated %s\n", expectedFile)
		return
	}

	archiveData, err := os.ReadFile(expectedFile)
	require.NoError(t, err, "expected.txtar not found in %s — run with -update to create it", dir)
	for _, f := range txtar.Parse(archiveData).Files {
		actual, err := os.ReadFile(filepath.Join(outDir, f.Name))
		require.NoError(t, err, "output file not found: %s", f.Name)
		assert.Equal(t, string(f.Data), string(actual), "mismatch in %s", f.Name)
	}
}

// Example e2e test, creates the input file based on values passed through env vars and checks that the
// expected files were generated.
// Would keep these as sanity tests in the short term and focus on mock tests for testing the generation logic, but we
// can consider:
//   - Setting up resources on which to run the generation as part of these tests.
//   - Running terraform validate/plan/appy on the generated config.
func TestE2E_SimpleProjectModule(t *testing.T) {
	projectID := os.Getenv("MONGODB_ATLAS_PROJECT_ID")
	require.NotEmpty(t, projectID, "MONGODB_ATLAS_PROJECT_ID env var must be set")

	// TODO: Will eventually move some of the following to helper functions as we create more e2e tests.

	inputHCL := fmt.Sprintf(`
		modules = ["project"]
		project_id = %q
	`, projectID)
	inputFile := filepath.Join(t.TempDir(), "input.tfvars")
	//nolint:gosec // path is constructed from t.TempDir(), no traversal risk
	require.NoError(t, os.WriteFile(inputFile, []byte(inputHCL), 0o600))

	outDir := t.TempDir()
	baseURL := atlasBaseURL()

	err := modulegen.Run(
		t.Context(),
		&modulegen.GenArgs{
			InputPath:    inputFile,
			OutputPath:   outDir,
			AtlasBaseURL: baseURL,
		},
		newE2EClient(t, baseURL),
	)
	require.NoError(t, err)

	// Check that all expected output files were generated.
	outputFiles := []string{"project.tf", "versions.tf", "variables.tf", "terraform.tfvars", "IMPORT_GUIDE.md"}
	for _, filename := range outputFiles {
		assert.FileExists(t, filepath.Join(outDir, filename))
	}
}

// newE2EClient creates a DefaultClient.
// When the `-record` flag is set to true, it wraps the client with a recordingClient that saves the ResourceStore
// to recordings/<testname_timestamp.json>.
func newE2EClient(t *testing.T, baseURL string) modulegen.Client {
	t.Helper()
	clientID := os.Getenv("MONGODB_ATLAS_CLIENT_ID")
	clientSecret := os.Getenv("MONGODB_ATLAS_CLIENT_SECRET")
	require.NotEmpty(t, clientID, "MONGODB_ATLAS_CLIENT_ID env var must be set")
	require.NotEmpty(t, clientSecret, "MONGODB_ATLAS_CLIENT_SECRET env var must be set")

	cfg := clientcredentials.NewConfig(clientID, clientSecret)
	cfg.TokenURL = baseURL + clientcredentials.TokenAPIPath
	cfg.RevokeURL = baseURL + clientcredentials.RevokeAPIPath
	httpClient := cfg.Client(context.Background())

	var client modulegen.Client = &modulegen.DefaultClient{
		HTTPClient:   httpClient,
		AtlasBaseURL: baseURL,
		UserAgent:    "atlas-cli-plugin-test", // TODO@non-spike: Revisit UserAgent value
	}
	if *record {
		savePath := fmt.Sprintf("recordings/%s_%d.json", t.Name(), time.Now().UnixMilli())
		client = &recordingClient{client: client, path: savePath}
	}
	return client
}

var _ modulegen.Client = &mockClient{}
var _ modulegen.Client = &recordingClient{}

// mockClient implements Client by unmarshalling an existing ResourceStore JSON file.
type mockClient struct {
	path string
}

func (m *mockClient) FetchResources(
	_ context.Context, _ *modulegen.Input, _ *modulegen.ResourcesToFetch,
) (*modulegen.ResourceStore, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil, fmt.Errorf("resource store file not found: %s — run with -record to create it: %w", m.path, err)
	}
	var store modulegen.ResourceStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ResourceStore from %s: %w", m.path, err)
	}
	return &store, nil
}

// recordingClient wraps a Client and marshals the ResourceStore it returns on FetchResources() to a JSON file.
type recordingClient struct {
	client modulegen.Client
	path   string
}

func (r *recordingClient) FetchResources(
	ctx context.Context,
	input *modulegen.Input,
	resourcesToFetch *modulegen.ResourcesToFetch,
) (*modulegen.ResourceStore, error) {
	store, err := r.client.FetchResources(ctx, input, resourcesToFetch)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ResourceStore: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create directory for ResourceStore: %w", err)
	}
	if err := os.WriteFile(r.path, data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write ResourceStore to %s: %w", r.path, err)
	}
	return store, nil
}
