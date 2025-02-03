package info

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/repository/state"
)

type InfoState struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Args []string
}

func (cmd *InfoState) Name() string {
	return "info_state"
}

func (cmd *InfoState) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.Args) == 0 {
		states, err := repo.GetStates()
		if err != nil {
			log.Fatal(err)
		}

		for _, state := range states {
			fmt.Printf("%x\n", state)
		}
	} else {
		for _, arg := range cmd.Args {
			// convert arg to [32]byte
			if len(arg) != 64 {
				log.Fatalf("invalid packfile hash: %s", arg)
			}

			b, err := hex.DecodeString(arg)
			if err != nil {
				log.Fatalf("invalid packfile hash: %s", arg)
			}

			// Convert the byte slice to a [32]byte
			var byteArray [32]byte
			copy(byteArray[:], b)

			rawStateRd, err := repo.GetState(byteArray)
			if err != nil {
				log.Fatal(err)
			}

			// Temporary scan cache to reconstruct that state.
			var identifier objects.Checksum
			n, err := rand.Read(identifier[:])
			if err != nil {
				return 1, err
			}
			if n != len(identifier) {
				return 1, io.ErrShortWrite
			}

			scanCache, err := repo.AppContext().GetCache().Scan(identifier)
			defer scanCache.Close()

			st, err := state.FromStream(rawStateRd, scanCache)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Version: %d.%d.%d\n", st.Metadata.Version/100, (st.Metadata.Version/10)%10, st.Metadata.Version%10)
			fmt.Printf("Creation: %s\n", st.Metadata.Timestamp)
			if len(st.Metadata.Extends) > 0 {
				fmt.Printf("Extends:\n")
				for _, stateID := range st.Metadata.Extends {
					fmt.Printf("  %x\n", stateID)
				}
			}

			printBlobs := func(name string, Type packfile.Type) {
				for snapshot, err := range st.ListObjectsOfType(Type) {
					if err != nil {
						fmt.Printf("Could not fetch blob entry for %s\n", name)
					} else {
						fmt.Printf("%s %x : packfile %x, offset %d, length %d\n",
							name,
							snapshot.Blob,
							snapshot.Location.Packfile,
							snapshot.Location.Offset,
							snapshot.Location.Length)
					}
				}
			}

			printBlobs("snapshot", packfile.TYPE_SNAPSHOT)
			printBlobs("chunk", packfile.TYPE_CHUNK)
			printBlobs("object", packfile.TYPE_OBJECT)
			printBlobs("file", packfile.TYPE_VFS)
			printBlobs("data", packfile.TYPE_DATA)
		}
	}
	return 0, nil
}
