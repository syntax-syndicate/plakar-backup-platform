package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/storage"
	_ "github.com/PlakarKorp/plakar/testing"
)

func TestNewRouter(t *testing.T) {
	repo := &repository.Repository{}
	token := "test-token"
	router := NewRouter(repo, token)
	if router == nil {
		t.Errorf("NewRouter returned a nil router")
	}
}

func TestAuthMiddleware(t *testing.T) {
	lstore, err := storage.NewStore("fs", "/test/location")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	ctx := appcontext.NewAppContext()
	ctx.SetCache(caching.NewManager("/tmp/test_plakar"))
	ctx.SetLogger(logging.NewLogger(os.Stdout, os.Stderr))
	repo, err := repository.New(ctx, lstore, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	token := "test-token"
	router := NewRouter(repo, token)
	if router == nil {
		t.Errorf("NewRouter returned a nil router")
	}

	req, err := http.NewRequest("GET", "/api/storage/configuration", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Invalid Token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code 401, got %d", w.Code)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w2.Code)
	}
}
