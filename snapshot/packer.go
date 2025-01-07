package snapshot

import (
	"bytes"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
)

type PackerMsg struct {
	Timestamp time.Time
	Type      packfile.Type
	Checksum  objects.Checksum
	Data      []byte
}

type Packer struct {
	Blobs    map[packfile.Type]map[[32]byte][]byte
	Packfile *packfile.PackFile
}

func NewPacker() *Packer {
	blobs := make(map[packfile.Type]map[[32]byte][]byte)
	for _, Type := range packfile.Types() {
		blobs[Type] = make(map[[32]byte][]byte)
	}
	return &Packer{
		Packfile: packfile.New(),
		Blobs:    blobs,
	}
}

func (packer *Packer) AddBlob(Type packfile.Type, checksum [32]byte, data []byte) {
	if _, ok := packer.Blobs[Type]; !ok {
		packer.Blobs[Type] = make(map[[32]byte][]byte)
	}
	packer.Blobs[Type][checksum] = data
	packer.Packfile.AddBlob(Type, checksum, data)
}

func (packer *Packer) Size() uint32 {
	return packer.Packfile.Size()
}

func (packer *Packer) Types() []packfile.Type {
	ret := make([]packfile.Type, 0, len(packer.Blobs))
	for k := range packer.Blobs {
		ret = append(ret, k)
	}
	return ret
}

func packerJob(snap *Snapshot) {
	wg := sync.WaitGroup{}
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var packer *Packer

			for msg := range snap.packerChan {
				if packer == nil {
					packer = NewPacker()
				}

				if msg, ok := msg.(*PackerMsg); !ok {
					panic("received data with unexpected type")
				} else {
					snap.Logger().Trace("packer", "%x: PackerMsg(%d, %064x), dt=%s", snap.Header.GetIndexShortID(), msg.Type, msg.Checksum, time.Since(msg.Timestamp))
					packer.AddBlob(msg.Type, msg.Checksum, msg.Data)
				}

				if packer.Size() > uint32(snap.repository.Configuration().Packfile.MaxSize) {
					err := snap.PutPackfile(packer)
					if err != nil {
						panic(err)
					}
					packer = nil
				}
			}

			if packer != nil {
				err := snap.PutPackfile(packer)
				if err != nil {
					panic(err)
				}
				packer = nil
			}
		}()
	}
	wg.Wait()
	snap.packerChanDone <- true
	close(snap.packerChanDone)
}

func (snap *Snapshot) PutBlob(Type packfile.Type, checksum [32]byte, data []byte) error {
	snap.Logger().Trace("snapshot", "%x: PutBlob(%d, %064x) len=%d", snap.Header.GetIndexShortID(), Type, checksum, len(data))

	encodedReader, err := snap.repository.Encode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	encoded, err := io.ReadAll(encodedReader)
	if err != nil {
		return err
	}

	snap.packerChan <- &PackerMsg{Type: Type, Timestamp: time.Now(), Checksum: checksum, Data: encoded}
	return nil
}

func (snap *Snapshot) GetBlob(Type packfile.Type, checksum [32]byte) ([]byte, error) {
	snap.Logger().Trace("snapshot", "%x: GetBlob(%x)", snap.Header.GetIndexShortID(), checksum)

	rd, err := snap.repository.GetBlob(Type, checksum)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(rd)
}

func (snap *Snapshot) BlobExists(Type packfile.Type, checksum [32]byte) bool {
	snap.Logger().Trace("snapshot", "%x: CheckBlob(%064x)", snap.Header.GetIndexShortID(), checksum)

	return snap.repository.BlobExists(Type, checksum)
}
