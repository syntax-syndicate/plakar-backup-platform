package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/PlakarKorp/plakar/storage"
)

type RepositoryInfoSnapshots struct {
	Total       int     `json:"total"`
	StorageSize int64   `json:"storage_size"`
	LogicalSize int64   `json:"logical_size"`
	Efficiency  float64 `json:"efficiency"`
}

type RepositoryInfoResponse struct {
	Location      string                  `json:"location"`
	Snapshots     RepositoryInfoSnapshots `json:"snapshots"`
	Configuration storage.Configuration   `json:"configuration"`
}

func repositoryInfo(w http.ResponseWriter, r *http.Request) error {
	configuration := lrepository.Configuration()
	nSnapshots, logicalSize, err := snapshot.LogicalSize(lrepository)
	if err != nil {
		return fmt.Errorf("unable to calculate logical size: %w", err)
	}

	efficiency := float64(0)
	storageSize := lrepository.StorageSize()
	if storageSize == -1 || logicalSize == 0 {
		efficiency = -1
	} else {
		usagePercent := (float64(storageSize) / float64(logicalSize)) * 100
		if usagePercent <= 100 {
			savings := 100 - usagePercent
			efficiency = savings
		} else {
			increase := usagePercent - 100
			if increase > 100 {
				efficiency = -1
			} else {
				efficiency = -1 * increase
			}
		}
	}

	return json.NewEncoder(w).Encode(Item[RepositoryInfoResponse]{Item: RepositoryInfoResponse{
		Location: lrepository.Location(),
		Snapshots: RepositoryInfoSnapshots{
			Total:       nSnapshots,
			StorageSize: int64(lrepository.StorageSize()),
			LogicalSize: logicalSize,
			Efficiency:  efficiency,
		},
		Configuration: configuration,
	}})
}

func repositorySnapshots(w http.ResponseWriter, r *http.Request) error {
	offset, err := QueryParamToUint32(r, "offset", 0, 0)
	if err != nil {
		return err
	}
	limit, err := QueryParamToUint32(r, "limit", 1, 50)
	if err != nil {
		return err
	}

	importerType, _, err := QueryParamToString(r, "importer")
	if err != nil {
		return err
	}

	sortKeys, err := QueryParamToSortKeys(r, "sort", "Timestamp")
	if err != nil {
		return err
	}

	lrepository.RebuildState()

	snapshotIDs, err := lrepository.GetSnapshots()
	if err != nil {
		return err
	}

	totalSnapshots := int(0)
	headers := make([]header.Header, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(lrepository, snapshotID)
		if err != nil {
			return err
		}

		if importerType != "" && strings.ToLower(snap.Header.GetSource(0).Importer.Type) != strings.ToLower(importerType) {
			snap.Close()
			continue
		}

		headers = append(headers, *snap.Header)
		totalSnapshots++
		snap.Close()
	}

	if limit == 0 {
		limit = uint32(len(headers))
	}

	header.SortHeaders(headers, sortKeys)
	if offset > uint32(len(headers)) {
		headers = []header.Header{}
	} else if offset+limit > uint32(len(headers)) {
		headers = headers[offset:]
	} else {
		headers = headers[offset : offset+limit]
	}

	items := Items[header.Header]{
		Total: totalSnapshots,
		Items: make([]header.Header, len(headers)),
	}
	for i, header := range headers {
		items.Items[i] = header
	}

	return json.NewEncoder(w).Encode(items)
}

func repositoryStates(w http.ResponseWriter, r *http.Request) error {
	states, err := lrepository.GetStates()
	if err != nil {
		return err
	}

	items := Items[objects.MAC]{
		Total: len(states),
		Items: make([]objects.MAC, len(states)),
	}
	for i, state := range states {
		items.Items[i] = state
	}

	return json.NewEncoder(w).Encode(items)
}

func repositoryState(w http.ResponseWriter, r *http.Request) error {
	stateBytes32, err := PathParamToID(r, "state")
	if err != nil {
		return err
	}

	_, rd, err := lrepository.GetState(stateBytes32)
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, rd); err != nil {
		log.Println("write failed:", err)
	}
	return nil
}

func repositoryImporterTypes(w http.ResponseWriter, r *http.Request) error {
	lrepository.RebuildState()

	snapshotIDs, err := lrepository.GetSnapshots()
	if err != nil {
		return err
	}

	importerTypesMap := make(map[string]struct{})
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(lrepository, snapshotID)
		if err != nil {
			return err
		}
		importerTypesMap[strings.ToLower(snap.Header.GetSource(0).Importer.Type)] = struct{}{}
	}

	importerTypes := make([]string, 0, len(importerTypesMap))
	for importerType := range importerTypesMap {
		importerTypes = append(importerTypes, importerType)
	}
	sort.Slice(importerTypes, func(i, j int) bool {
		return importerTypes[i] < importerTypes[j]
	})

	type Entry struct {
		Name string `json:"name"`
	}

	items := Items[Entry]{
		Total: len(importerTypes),
		Items: make([]Entry, len(importerTypes)),
	}
	for i, importerType := range importerTypes {
		items.Items[i] = Entry{Name: importerType}
	}

	return json.NewEncoder(w).Encode(items)
}

type TimelineLocation struct {
	Snapshot header.Header `json:"snapshot"`
	Entry    vfs.Entry     `json:"vfs_entry"`
}

func repositoryLocatePathname(w http.ResponseWriter, r *http.Request) error {
	offset, err := QueryParamToUint32(r, "offset", 0, 0)
	if err != nil {
		return err
	}
	limit, err := QueryParamToUint32(r, "limit", 1, 50)
	if err != nil {
		return err
	}

	importerType, _, err := QueryParamToString(r, "importerType")
	if err != nil {
		return err
	}

	importerOrigin, _, err := QueryParamToString(r, "importerOrigin")
	if err != nil {
		return err
	}

	resource, _, err := QueryParamToString(r, "resource")
	if err != nil {
		return err
	}

	sortKeys, err := QueryParamToSortKeys(r, "sort", "Timestamp")
	if err != nil {
		return err
	}

	lrepository.RebuildState()

	snapshotIDs, err := lrepository.GetSnapshots()
	if err != nil {
		return err
	}

	totalSnapshots := int(0)
	locations := make([]TimelineLocation, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(lrepository, snapshotID)
		if err != nil {
			return err
		}

		if importerType != "" && !strings.EqualFold(snap.Header.GetSource(0).Importer.Type, importerType) {
			snap.Close()
			continue
		}

		if importerOrigin != "" && !strings.EqualFold(snap.Header.GetSource(0).Importer.Origin, importerOrigin) {
			snap.Close()
			continue
		}

		pvfs, err := snap.Filesystem()
		if err != nil {
			snap.Close()
			continue
		}

		entry, err := pvfs.GetEntry(resource)
		if err != nil {
			snap.Close()
			continue
		}

		locations = append(locations, TimelineLocation{
			Snapshot: *snap.Header,
			Entry:    *entry,
		})
		totalSnapshots++
		snap.Close()
	}

	if limit == 0 {
		limit = uint32(len(locations))
	}

	sortFunc := func(a, b TimelineLocation) int {
		if a.Snapshot.Timestamp.Before(b.Snapshot.Timestamp) {
			return -1
		}
		if a.Snapshot.Timestamp.After(b.Snapshot.Timestamp) {
			return 1
		}
		return 0
	}

	if len(sortKeys) > 0 {
		switch sortKeys[0] {
		case "-Timestamp":
			sortFunc = func(a, b TimelineLocation) int {
				if a.Snapshot.Timestamp.After(b.Snapshot.Timestamp) {
					return -1
				}
				if a.Snapshot.Timestamp.Before(b.Snapshot.Timestamp) {
					return 1
				}
				return 0
			}
		}
	}

	slices.SortFunc(locations, sortFunc)

	if offset > uint32(len(locations)) {
		locations = []TimelineLocation{}
	} else if offset+limit > uint32(len(locations)) {
		locations = locations[offset:]
	} else {
		locations = locations[offset : offset+limit]
	}

	items := Items[TimelineLocation]{
		Total: totalSnapshots,
		Items: make([]TimelineLocation, 0, len(locations)),
	}
	items.Items = append(items.Items, locations...)

	return json.NewEncoder(w).Encode(items)
}

// XXXXX

type TokenResponse struct {
	Token string `json:"token"`
}

type LoginRequestGithub struct {
	Redirect string `json:"redirect"`
}

type LoginRequestEmail struct {
	Email    string `json:"email"`
	Redirect string `json:"redirect"`
}

func repositoryLoginGithub(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequestGithub

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	parameters := make(map[string]string)
	parameters["redirect"] = req.Redirect

	lf, err := utils.NewLoginFlow(lrepository.AppContext(), lrepository.Configuration().RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to create login flow: %w", err)
	}

	redirectURL, err := lf.RunUI("github", parameters)
	if err != nil {
		return fmt.Errorf("failed to run login flow: %w", err)
	}

	ret := struct {
		URL string `json:"URL"`
	}{
		URL: redirectURL,
	}

	return json.NewEncoder(w).Encode(ret)
}

func repositoryLoginEmail(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequestEmail

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	parameters := make(map[string]string)
	parameters["email"] = req.Email
	parameters["redirect"] = req.Redirect

	lf, err := utils.NewLoginFlow(lrepository.AppContext(), lrepository.Configuration().RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to create login flow: %w", err)
	}

	redirectURL, err := lf.RunUI("email", parameters)
	if err != nil {
		return fmt.Errorf("failed to run login flow: %w", err)
	}

	ret := struct {
		URL string `json:"URL"`
	}{
		URL: redirectURL,
	}
	return json.NewEncoder(w).Encode(ret)
}

func repositoryLogout(w http.ResponseWriter, r *http.Request) error {
	configuration := lrepository.Configuration()
	if cache, err := lrepository.AppContext().GetCache().Repository(configuration.RepositoryID); err != nil {
		return err
	} else if exists := cache.HasAuthToken(); !exists {
		return nil
	} else {
		return cache.DeleteAuthToken()
	}
}
