package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"

	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/storage"
)

var lstore storage.Store
var lconfig storage.Configuration
var lrepository *repository.Repository

type Item[T any] struct {
	Item T `json:"item"`
}

type Items[T any] struct {
	Total int `json:"total"`
	Items []T `json:"items"`
}

type ItemsPage[T any] struct {
	HasNext bool `json:"has_next"`
	Items   []T  `json:"items"`
}

type ApiErrorRes struct {
	Error *ApiError `json:"error"`
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, repository.ErrNotReadable):
		err = &ApiError{
			HttpCode: 400,
			ErrCode:  "bad-request",
			Message:  err.Error(),
		}
	case errors.Is(err, repository.ErrBlobNotFound):
		fallthrough
	case errors.Is(err, repository.ErrPackfileNotFound):
		fallthrough
	case errors.Is(err, fs.ErrNotExist):
		fallthrough
	case errors.Is(err, snapshot.ErrNotFound):
		err = &ApiError{
			HttpCode: 404,
			ErrCode:  "not-found",
			Message:  err.Error(),
		}
	}

	apierr, ok := err.(*ApiError)
	if !ok {
		log.Printf("Unknown error encountered while serving %s: %v", r.URL, err)
		apierr = &ApiError{
			HttpCode: 500,
			ErrCode:  "internal-error",
			Message:  "Internal server error. Check server logs for more information.",
		}
	}

	w.WriteHeader(apierr.HttpCode)
	_ = json.NewEncoder(w).Encode(&ApiErrorRes{apierr})
}

type JSONAPIView func(w http.ResponseWriter, r *http.Request) error

func (view JSONAPIView) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := view(w, r); err != nil {
		handleError(w, r, err)
	}
}

type APIView func(w http.ResponseWriter, r *http.Request) error

func (view APIView) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := view(w, r); err != nil {
		w.Header().Set("Content-Type", "application/json")
		handleError(w, r, err)
	}
}

// TokenAuthMiddleware is a middleware that checks for the token in the request. If the token is empty, the middleware is a no-op.
func TokenAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token != "" {
				key := r.Header.Get("Authorization")
				if key == "" {
					handleError(w, r, authError("missing Authorization header"))
					return
				}

				if subtle.ConstantTimeCompare([]byte(key), []byte("Bearer "+token)) == 0 {
					handleError(w, r, authError("invalid token"))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func apiInfo(w http.ResponseWriter, r *http.Request) error {
	res := &struct {
		Version   string `json:"version"`
		Browsable bool   `json:"browsable"`
	}{
		Version:   utils.GetVersion(),
		Browsable: lrepository.Store().Mode()&storage.ModeRead != 0,
	}
	return json.NewEncoder(w).Encode(res)
}

func SetupRoutes(server *http.ServeMux, repo *repository.Repository, token string) {
	lstore = repo.Store()
	lconfig = repo.Configuration()
	lrepository = repo

	authToken := TokenAuthMiddleware(token)
	urlSigner := NewSnapshotReaderURLSigner(token)

	// Catch all API endpoint, called if no more specific API endpoint is found
	server.Handle("/api/", JSONAPIView(func(w http.ResponseWriter, r *http.Request) error {
		return &ApiError{
			HttpCode: 404,
			ErrCode:  "not-found",
			Message:  "API endpoint not found",
		}
	}))

	server.Handle("GET /api/info", authToken(JSONAPIView(apiInfo)))

	server.Handle("GET /api/repository/info", authToken(JSONAPIView(repositoryInfo)))
	server.Handle("GET /api/repository/snapshots", authToken(JSONAPIView(repositorySnapshots)))
	server.Handle("GET /api/repository/locate-pathname", authToken(JSONAPIView(repositoryLocatePathname)))
	server.Handle("GET /api/repository/importer-types", authToken(JSONAPIView(repositoryImporterTypes)))
	server.Handle("GET /api/repository/states", authToken(JSONAPIView(repositoryStates)))
	server.Handle("GET /api/repository/state/{state}", authToken(JSONAPIView(repositoryState)))

	server.Handle("GET /api/snapshot/{snapshot}", authToken(JSONAPIView(snapshotHeader)))
	server.Handle("GET /api/snapshot/reader/{snapshot_path...}", urlSigner.VerifyMiddleware(APIView(snapshotReader)))
	server.Handle("POST /api/snapshot/reader-sign-url/{snapshot_path...}", authToken(JSONAPIView(urlSigner.Sign)))

	server.Handle("GET /api/snapshot/vfs/{snapshot_path...}", authToken(JSONAPIView(snapshotVFSBrowse)))
	server.Handle("GET /api/snapshot/vfs/children/{snapshot_path...}", authToken(JSONAPIView(snapshotVFSChildren)))
	server.Handle("GET /api/snapshot/vfs/chunks/{snapshot_path...}", authToken(JSONAPIView(snapshotVFSChunks)))
	server.Handle("GET /api/snapshot/vfs/search/{snapshot_path...}", authToken(JSONAPIView(snapshotVFSSearch)))
	server.Handle("GET /api/snapshot/vfs/errors/{snapshot_path...}", authToken(JSONAPIView(snapshotVFSErrors)))

	server.Handle("POST /api/snapshot/vfs/downloader/{snapshot_path...}", authToken(JSONAPIView(snapshotVFSDownloader)))
	server.Handle("GET /api/snapshot/vfs/downloader-sign-url/{id}", JSONAPIView(snapshotVFSDownloaderSigned))
}
