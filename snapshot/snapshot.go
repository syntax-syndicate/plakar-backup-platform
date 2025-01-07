package snapshot

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/PlakarKorp/plakar/context"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/repository/state"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("snapshot not found")
)

type Snapshot struct {
	repository *repository.Repository
	stateDelta *state.State

	filesystem *vfs.Filesystem

	SkipDirs []string

	Header *header.Header

	packerChan     chan interface{}
	packerChanDone chan bool
}

func New(repo *repository.Repository) (*Snapshot, error) {
	var identifier objects.Checksum

	n, err := rand.Read(identifier[:])
	if err != nil {
		return nil, err
	}
	if n != len(identifier) {
		return nil, io.ErrShortWrite
	}

	snap := &Snapshot{
		repository: repo,
		stateDelta: repo.NewStateDelta(),

		Header: header.NewHeader("default", identifier),

		packerChan:     make(chan interface{}, runtime.NumCPU()*2+1),
		packerChanDone: make(chan bool),
	}

	if snap.Context().GetIdentity() != uuid.Nil {
		snap.Header.Identity.Identifier = snap.Context().GetIdentity()
		snap.Header.Identity.PublicKey = snap.Context().GetKeypair().PublicKey
	}

	snap.Header.SetContext("Hostname", snap.Context().GetHostname())
	snap.Header.SetContext("Username", snap.Context().GetUsername())
	snap.Header.SetContext("OperatingSystem", snap.Context().GetOperatingSystem())
	snap.Header.SetContext("MachineID", snap.Context().GetMachineID())
	snap.Header.SetContext("CommandLine", snap.Context().GetCommandLine())
	snap.Header.SetContext("ProcessID", fmt.Sprintf("%d", snap.Context().GetProcessID()))
	snap.Header.SetContext("Architecture", snap.Context().GetArchitecture())
	snap.Header.SetContext("NumCPU", fmt.Sprintf("%d", runtime.NumCPU()))
	snap.Header.SetContext("GOMAXPROCS", fmt.Sprintf("%d", runtime.GOMAXPROCS(0)))
	snap.Header.SetContext("Client", snap.Context().GetPlakarClient())

	go packerJob(snap)

	repo.Logger().Trace("snapshot", "%x: New()", snap.Header.GetIndexShortID())
	return snap, nil
}

func Load(repo *repository.Repository, Identifier objects.Checksum) (*Snapshot, error) {
	hdr, _, err := GetSnapshot(repo, Identifier)
	if err != nil {
		return nil, err
	}

	snapshot := &Snapshot{}
	snapshot.repository = repo
	snapshot.Header = hdr

	repo.Logger().Trace("snapshot", "%x: Load()", snapshot.Header.GetIndexShortID())
	return snapshot, nil
}

func Clone(repo *repository.Repository, Identifier objects.Checksum) (*Snapshot, error) {
	snap, err := Load(repo, Identifier)
	if err != nil {
		return nil, err
	}
	snap.Header.Timestamp = time.Now()

	uuidBytes, err := uuid.Must(uuid.NewRandom()).MarshalBinary()
	if err != nil {
		return nil, err
	}

	snap.stateDelta = state.New()

	snap.Header.Identifier = repo.Checksum(uuidBytes[:])
	snap.packerChan = make(chan interface{}, runtime.NumCPU()*2+1)
	snap.packerChanDone = make(chan bool)
	go packerJob(snap)

	repo.Logger().Trace("snapshot", "%x: Clone(): %s", snap.Header.Identifier, snap.Header.GetIndexShortID())
	return snap, nil
}

func Fork(repo *repository.Repository, Identifier objects.Checksum) (*Snapshot, error) {
	var identifier objects.Checksum

	n, err := rand.Read(identifier[:])
	if err != nil {
		return nil, err
	}
	if n != len(identifier) {
		return nil, io.ErrShortWrite
	}

	snap, err := Clone(repo, Identifier)
	if err != nil {
		return nil, err
	}

	snap.Header.Identifier = identifier

	snap.Logger().Trace("snapshot", "%x: Fork(): %s", snap.Header.Identifier, snap.Header.GetIndexShortID())
	return snap, nil
}

func (snap *Snapshot) Close() error {
	snap.Logger().Trace("snapshot", "%x: Close(): %s", snap.Header.Identifier, snap.Header.GetIndexShortID())
	return nil
}

func (snap *Snapshot) Context() *context.Context {
	return snap.Repository().Context()
}

func (snap *Snapshot) Event(evt events.Event) {
	snap.Context().Events().Send(evt)
}

func GetSnapshot(repo *repository.Repository, Identifier objects.Checksum) (*header.Header, bool, error) {
	repo.Logger().Trace("snapshot", "repository.GetSnapshot(%x)", Identifier)

	rd, err := repo.GetBlob(packfile.TYPE_SNAPSHOT, Identifier)
	if err != nil {
		if errors.Is(err, repository.ErrBlobNotFound) {
			err = ErrNotFound
		}
		return nil, false, err
	}

	buffer, err := io.ReadAll(rd)
	if err != nil {
		return nil, false, err
	}

	hdr, err := header.NewFromBytes(buffer)
	if err != nil {
		return nil, false, err
	}

	return hdr, false, nil
}

func (snap *Snapshot) Repository() *repository.Repository {
	return snap.repository
}

func (snap *Snapshot) LookupObject(checksum objects.Checksum) (*objects.Object, error) {
	buffer, err := snap.GetBlob(packfile.TYPE_OBJECT, checksum)
	if err != nil {
		return nil, err
	}
	return objects.NewObjectFromBytes(buffer)
}

func (snap *Snapshot) ListChunks() (<-chan objects.Checksum, error) {
	fs, err := snap.Filesystem()
	if err != nil {
		return nil, err
	}
	c := make(chan objects.Checksum)
	go func() {
		for filename := range fs.Files() {
			fsentry, err := fs.GetEntry(filename)
			if err != nil {
				break
			}
			if fsentry.Object == nil {
				continue
			}
			for _, chunk := range fsentry.Object.Chunks {
				c <- chunk.Checksum
			}
		}
		close(c)
	}()
	return c, nil
}

func (snap *Snapshot) ListObjects() (<-chan objects.Checksum, error) {
	fs, err := snap.Filesystem()
	if err != nil {
		return nil, err
	}
	c := make(chan objects.Checksum)
	go func() {
		for filename := range fs.Files() {
			fsentry, err := fs.GetEntry(filename)
			if err != nil {
				break
			}
			if fsentry.Object == nil {
				continue
			}
			c <- fsentry.Object.Checksum
		}
		close(c)
	}()
	return c, nil
}

func (snap *Snapshot) ListDatas() (<-chan objects.Checksum, error) {
	c := make(chan objects.Checksum)

	go func() {
		c <- snap.Header.Metadata
		close(c)
	}()

	return c, nil
}

func (snap *Snapshot) Logger() *logging.Logger {
	return snap.Context().GetLogger()
}
