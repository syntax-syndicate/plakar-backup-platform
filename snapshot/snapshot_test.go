package snapshot_test

import (
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func generateSnapshot(t *testing.T) (*repository.Repository, *snapshot.Snapshot) {
	repo := ptesting.GenerateRepository(t, nil, nil, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("dummy.txt", 0644, "hello"),
	})
	return repo, snap
}

func TestSnapshot(t *testing.T) {
	repo, snap := generateSnapshot(t)
	defer snap.Close()

	appCtx := snap.AppContext()
	require.NotNil(t, appCtx)
	require.NotNil(t, appCtx.GetCache())
	defer appCtx.GetCache().Close()

	events := appCtx.Events()
	require.NotNil(t, events)

	snapFs, err := snap.Filesystem()
	require.NoError(t, err)
	require.NotNil(t, snapFs)

	snap2, err := snapshot.Load(repo, snap.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap2)

	require.Equal(t, snap.Header.Identifier, snap2.Header.Identifier)
	require.Equal(t, snap.Header.Timestamp.Truncate(time.Nanosecond), snap2.Header.Timestamp.Truncate(time.Nanosecond))
}

func checkmacs(t *testing.T, snap *snapshot.Snapshot) bool {
	var (
		indexContent   = map[objects.MACTuple]struct{}{}
		crawledContent = map[objects.MACTuple]struct{}{}
	)

	var err error
	for res, mac := range snap.MACs(&err) {
		indexContent[objects.MACTuple{Resource: res, MAC: mac}] = struct{}{}
	}
	if err != nil {
		t.Fatalf("snap.MACs failed with %v", err)
	}

	for res, mac := range snap.CrawlMACs(&err) {
		crawledContent[objects.MACTuple{Resource: res, MAC: mac}] = struct{}{}
	}
	if err != nil {
		t.Fatalf("snap.MACs failed with %v", err)
	}

	success := true
	for tupl := range indexContent {
		if _, ok := crawledContent[tupl]; !ok {
			success = false
			t.Errorf("MAC index yield but not found in crawling: %v %x",
				tupl.Resource.String(), tupl.MAC)
		}
	}

	for tupl := range crawledContent {
		if _, ok := indexContent[tupl]; !ok {
			success = false
			t.Errorf("crawling yield but not found in MAC index: %v %x",
				tupl.Resource.String(), tupl.MAC)
		}
	}

	return success
}

func TestMACs(t *testing.T) {
	repo := ptesting.GenerateRepository(t, nil, nil, nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("dummy.txt", 0644, "hello"),
	})
	defer snap1.Close()

	if !checkmacs(t, snap1) {
		t.Errorf("MACs and CrawlMACs don't agree!")
	}

	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("dummy.txt", 0644, "hello"),
		ptesting.NewMockFile("foo.txt", 0644, "foo bar baz"),
	})
	defer snap2.Close()

	if !checkmacs(t, snap2) {
		t.Errorf("MACs and CrawlMACs don't agree after incremental backup")
	}
}
