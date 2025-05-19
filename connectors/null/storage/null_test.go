package null

import (
	"bytes"
	"io"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/stretchr/testify/require"
)

func TestNullBackend(t *testing.T) {
	ctx := appcontext.NewAppContext()

	// create a repository
	repo, err := NewStore(ctx, map[string]string{"location": "/test/location"})
	if err != nil {
		t.Fatal("error creating repository", err)
	}

	location := repo.Location()
	require.Equal(t, "/test/location", location)

	config := storage.NewConfiguration()
	serializedConfig, err := config.ToBytes()
	require.NoError(t, err)

	err = repo.Create(ctx, serializedConfig)
	require.NoError(t, err)

	_, err = repo.Open(ctx)
	require.NoError(t, err)
	// only test one field
	//require.Equal(t, repo.Configuration().Version, versioning.FromString(storage.VERSION))

	err = repo.Close()
	require.NoError(t, err)

	mac := objects.MAC{0x10}

	// states
	macs, err := repo.GetStates()
	require.NoError(t, err)
	require.Equal(t, macs, []objects.MAC{})

	_, err = repo.PutState(mac, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	rd, err := repo.GetState(mac)
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "", buf.String())

	err = repo.DeleteState(mac)
	require.NoError(t, err)

	// packfiles
	macs, err = repo.GetPackfiles()
	require.NoError(t, err)
	require.Equal(t, macs, []objects.MAC{})

	_, err = repo.PutPackfile(mac, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	rd, err = repo.GetPackfile(mac)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "", buf.String())

	rd, err = repo.GetPackfileBlob(mac, 0, 0)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "", buf.String())

	err = repo.DeletePackfile(mac)
	require.NoError(t, err)
}
