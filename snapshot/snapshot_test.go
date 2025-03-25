package snapshot_test

import (
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/encryption/keypair"
	"github.com/PlakarKorp/plakar/snapshot"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func generateSnapshot(t *testing.T, keyPair *keypair.KeyPair) *snapshot.Snapshot {
	return ptesting.GenerateSnapshot(t, nil, nil, keyPair, []ptesting.MockFile{
		ptesting.NewMockFile("dummy.txt", 0644, "hello"),
	})
}

func TestSnapshot(t *testing.T) {
	snap := generateSnapshot(t, nil)
	defer snap.Close()

	appCtx := snap.AppContext()
	require.NotNil(t, appCtx)
	require.NotNil(t, appCtx.GetCache())
	defer appCtx.GetCache().Close()

	events := appCtx.Events()
	require.NotNil(t, events)

	err := snap.Repository().RebuildState()
	require.NoError(t, err)

	snapFs, err := snap.Filesystem()
	require.NoError(t, err)
	require.NotNil(t, snapFs)

	snap2, err := snapshot.Load(snap.Repository(), snap.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap2)

	require.Equal(t, snap.Header.Identifier, snap2.Header.Identifier)
	require.Equal(t, snap.Header.Timestamp.Truncate(time.Nanosecond), snap2.Header.Timestamp.Truncate(time.Nanosecond))

	snap3, err := snapshot.Clone(snap.Repository(), snap.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap3)

	require.NotEqual(t, snap.Header.Identifier, snap3.Header.Identifier)

	snap4, err := snapshot.Fork(snap.Repository(), snap.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap4)

	require.NotEqual(t, snap.Header.Identifier, snap4.Header.Identifier)
}
