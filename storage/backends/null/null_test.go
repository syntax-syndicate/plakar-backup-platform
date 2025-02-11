package null

import (
	"bytes"
	"io"
	"testing"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/stretchr/testify/require"
)

func TestNullBackend(t *testing.T) {
	// create a repository
	repo := NewRepository("/test/location")
	if repo == nil {
		t.Fatal("error creating repository")
	}

	location := repo.Location()
	require.Equal(t, "/test/location", location)

	config := storage.NewConfiguration()
	serializedConfig, err := config.ToBytes()
	require.NoError(t, err)

	err = repo.Create(location, serializedConfig)
	require.NoError(t, err)

	_, err = repo.Open(location)
	require.NoError(t, err)
	// only test one field
	//require.Equal(t, repo.Configuration().Version, versioning.FromString(storage.VERSION))

	err = repo.Close()
	require.NoError(t, err)

	// snapshots
	r, ok := repo.(*Repository)
	require.True(t, ok)
	snaps, err := r.GetSnapshots()
	require.NoError(t, err)
	require.Equal(t, snaps, []objects.MAC{})

	mac := objects.MAC{0x10}
	err = r.PutSnapshot(mac, []byte("test"))
	require.NoError(t, err)

	retrievedSnapshot, err := r.GetSnapshot(mac)
	require.NoError(t, err)
	require.Equal(t, []byte(""), retrievedSnapshot)

	err = r.DeleteSnapshot(mac)
	require.NoError(t, err)

	// states
	macs, err := repo.GetStates()
	require.NoError(t, err)
	require.Equal(t, macs, []objects.MAC{})

	err = repo.PutState(mac, bytes.NewReader([]byte("test")))
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

	err = repo.PutPackfile(mac, bytes.NewReader([]byte("test")))
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
