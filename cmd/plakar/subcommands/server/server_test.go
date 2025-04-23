package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/network"
	"github.com/PlakarKorp/plakar/resources"
	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	"github.com/PlakarKorp/plakar/storage"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdServerDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	snap.Close()

	ctx := repo.AppContext()
	args := []string{"-listen", "127.0.0.1:12345"}

	subcommand := &Server{}
	err := subcommand.Parse(ctx, args)

	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "server", subcommand.Name())

	subCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx.SetContext(subCtx)

	go func() {
		status, err := subcommand.Execute(ctx, repo)
		require.NoError(t, err)
		require.Equal(t, 0, status)
	}()

	// wait for the server to start
	time.Sleep(100 * time.Millisecond)

	req, err := http.NewRequest("GET", "http://localhost:12345/", bytes.NewBuffer([]byte(`{"Repository": ""}`)))
	require.NoError(t, err, "creating request")

	client := &http.Client{}
	w := httptest.NewRecorder()
	require.Equal(t, http.StatusOK, w.Code, "expected status code 200")

	response, err := client.Do(req)
	require.NoError(t, err, "making request")

	rawBody, err := io.ReadAll(response.Body)
	require.NoError(t, err, "reading response")

	var resOpen network.ResOpen
	err = json.Unmarshal(rawBody, &resOpen)
	require.NoError(t, err, "unmarshaling response")

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	version, unwrappedConfigRd, err := storage.Deserialize(hasher, resources.RT_CONFIG, bytes.NewReader(resOpen.Configuration))
	require.NoError(t, err, "deserializing configuration")

	unwrappedConfig, err := io.ReadAll(unwrappedConfigRd)
	require.NoError(t, err, "reading deserializing configuration")

	configInstance, err := storage.NewConfigurationFromBytes(version, unwrappedConfig)
	require.NoError(t, err, "reading configuration")

	// we dont test all the field from configuration
	require.Equal(t, versioning.FromString(storage.VERSION), configInstance.Version)
}
