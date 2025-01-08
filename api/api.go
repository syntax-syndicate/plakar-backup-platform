package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/storage"
)

var lstore storage.Store
var lrepository *repository.Repository

type Item struct {
	Item interface{} `json:"item"`
}

type Items struct {
	Total int           `json:"total"`
	Items []interface{} `json:"items"`
}

type ApiErrorRes struct {
	Error *ApiError `json:"error"`
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrBlobNotFound):
		fallthrough
	case errors.Is(err, repository.ErrPackfileNotFound):
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
		apierr = &ApiError{
			HttpCode: 500,
			ErrCode:  "internal-error",
			Message:  "Internal server error. Check server logs for more information.",
		}
	}

	w.WriteHeader(apierr.HttpCode)
	_ = json.NewEncoder(w).Encode(&ApiErrorRes{apierr})
}

type APIView struct {
	fn     func(w http.ResponseWriter, r *http.Request) error
	method string
}

func (view APIView) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	expectedMethod := view.method
	if expectedMethod == "" {
		expectedMethod = http.MethodGet
	}

	if r.Method != expectedMethod {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := view.fn(w, r); err != nil {
		handleError(w, err)
	}
}

// TokenAuthMiddleware is a middleware that checks for the token in the request. If the token is empty, the middleware is a no-op.
func TokenAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token != "" {
				key := r.Header.Get("Authorization")
				if key == "" {
					handleError(w, authError("missing Authorization header"))
					return
				}

				if strings.Compare(key, "Bearer "+token) != 0 {
					handleError(w, authError("invalid token"))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func SetupRoutes(server *http.ServeMux, repo *repository.Repository, token string) {
	lstore = repo.Store()
	lrepository = repo

	authToken := TokenAuthMiddleware(token)
	urlSigner := NewSnapshotReaderURLSigner(token)

	server.Handle("/api/storage/configuration", authToken(APIView{fn: storageConfiguration}))
	server.Handle("/api/storage/states", authToken(APIView{fn: storageStates}))
	server.Handle("/api/storage/state/{state}", authToken(APIView{fn: storageState}))
	server.Handle("/api/storage/packfiles", authToken(APIView{fn: storagePackfiles}))
	server.Handle("/api/storage/packfile/{packfile}", authToken(APIView{fn: storagePackfile}))

	server.Handle("/api/repository/configuration", authToken(APIView{fn: repositoryConfiguration}))
	server.Handle("/api/repository/snapshots", authToken(APIView{fn: repositorySnapshots}))
	server.Handle("/api/repository/states", authToken(APIView{fn: repositoryStates}))
	server.Handle("/api/repository/state/{state}", authToken(APIView{fn: repositoryState}))
	server.Handle("/api/repository/packfiles", authToken(APIView{fn: repositoryPackfiles}))
	server.Handle("/api/repository/packfile/{packfile}", authToken(APIView{fn: repositoryPackfile}))

	server.Handle("/api/snapshot/{snapshot}", authToken(APIView{fn: snapshotHeader}))
	server.Handle("/api/snapshot/reader/{snapshot_path...}", urlSigner.VerifyMiddleware(APIView{fn: snapshotReader}))
	server.Handle("/api/snapshot/reader-sign-url/{snapshot_path...}", authToken(APIView{fn: urlSigner.Sign, method: http.MethodPost}))

	server.Handle("/api/snapshot/search/{snapshot_path...}", authToken(APIView{fn: snapshotSearch}))
	server.Handle("/api/snapshot/vfs/{snapshot_path...}", authToken(APIView{fn: snapshotVFSBrowse}))
	server.Handle("/api/snapshot/vfs/children/{snapshot_path...}", authToken(APIView{fn: snapshotVFSChildren}))
	server.Handle("/api/snapshot/vfs/errors/{snapshot_path...}", authToken(APIView{fn: snapshotVFSErrors}))
}
