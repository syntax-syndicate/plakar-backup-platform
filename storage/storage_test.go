package storage_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/storage"
	_ "github.com/PlakarKorp/plakar/testing"
)

func TestNewStore(t *testing.T) {
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(os.Stdout, os.Stderr))
	ctx.MaxConcurrency = runtime.NumCPU()*8 + 1

	store, err := storage.NewStore("fs", "/test/location")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if store.Location() != "/test/location" {
		t.Errorf("expected location to be '/test/location', got %v", store.Location())
	}
}

func TestCreateStore(t *testing.T) {
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(os.Stdout, os.Stderr))
	ctx.MaxConcurrency = runtime.NumCPU()*8 + 1

	config := storage.NewConfiguration()
	store, err := storage.Create("/test/location", *config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if store.Configuration().RepositoryID != config.RepositoryID {
		t.Errorf("expected RepositoryID to match, got %v and %v", store.Configuration().RepositoryID, config.RepositoryID)
	}
}

func TestOpenStore(t *testing.T) {
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(os.Stdout, os.Stderr))
	ctx.MaxConcurrency = runtime.NumCPU()*8 + 1

	store, err := storage.Open("/test/location")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if store.Location() != "/test/location" {
		t.Errorf("expected location to be '/test/location', got %v", store.Location())
	}
}
