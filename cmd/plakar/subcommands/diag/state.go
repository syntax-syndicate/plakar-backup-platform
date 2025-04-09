package diag

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/repository/state"
	"github.com/PlakarKorp/plakar/resources"
)

type DiagState struct {
	RepositorySecret []byte

	Args []string

	Locate string
}

func (cmd *DiagState) Name() string {
	return "diag_state"
}

func (cmd *DiagState) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.Args) == 0 {
		states, err := repo.GetStates()
		if err != nil {
			return 1, err
		}

		for _, st := range states {
			if cmd.Locate != "" {
				version, rawStateRd, err := repo.GetState(st)
				if err != nil {
					return 1, err
				}

				// Temporary scan cache to reconstruct that state.
				identifier := objects.RandomMAC()
				scanCache, err := repo.AppContext().GetCache().Scan(identifier)
				if err != nil {
					return 1, err
				}
				defer scanCache.Close()

				st, err := state.FromStream(version, rawStateRd, scanCache)
				if err != nil {
					return 1, err
				}

				printBlobs := func(name string, Type resources.Type) {
					for snapshot, err := range st.ListObjectsOfType(Type) {
						if err != nil {
							fmt.Fprintf(ctx.Stdout, "Could not fetch blob entry for %s\n", name)
						} else {
							if strings.Contains(fmt.Sprintf("%x", snapshot.Blob), cmd.Locate) {
								fmt.Fprintf(ctx.Stdout, "state=%x: %s %x : packfile %x, offset %d, length %d\n",
									st,
									name,
									snapshot.Blob,
									snapshot.Location.Packfile,
									snapshot.Location.Offset,
									snapshot.Location.Length)
							}
						}
					}
				}
				printDeleted := func(name string, Type resources.Type) {
					for deletedEntry, err := range st.ListDeletedResources(Type) {
						if err != nil {
							fmt.Fprintf(ctx.Stdout, "Could not fetch deleted blob entry for %s\n", name)
						} else {
							if strings.Contains(fmt.Sprintf("%x", deletedEntry.Blob), cmd.Locate) {
								fmt.Fprintf(ctx.Stdout, "state=%x, deleted %s: %x, when=%s\n",
									st,
									name,
									deletedEntry.Blob,
									deletedEntry.When)
							}
						}
					}
				}

				for _, Type := range resources.Types() {
					printDeleted(Type.String(), Type)
					printBlobs(Type.String(), Type)
				}
			} else {
				fmt.Fprintf(ctx.Stdout, "%x\n", st)
			}
		}
	} else {
		for _, arg := range cmd.Args {
			// convert arg to [32]byte
			if len(arg) != 64 {
				return 1, fmt.Errorf("invalid packfile hash: %s", arg)
			}

			b, err := hex.DecodeString(arg)
			if err != nil {
				return 1, fmt.Errorf("invalid packfile hash: %s", arg)
			}

			// Convert the byte slice to a [32]byte
			var byteArray [32]byte
			copy(byteArray[:], b)

			version, rawStateRd, err := repo.GetState(byteArray)
			if err != nil {
				return 1, err
			}

			// Temporary scan cache to reconstruct that state.
			identifier := objects.RandomMAC()
			scanCache, err := repo.AppContext().GetCache().Scan(identifier)
			if err != nil {
				return 1, err
			}
			defer scanCache.Close()

			st, err := state.FromStream(version, rawStateRd, scanCache)
			if err != nil {
				return 1, err
			}

			fmt.Fprintf(ctx.Stdout, "Version: %s\n", st.Metadata.Version)
			fmt.Fprintf(ctx.Stdout, "Creation: %s\n", st.Metadata.Timestamp)
			fmt.Fprintf(ctx.Stdout, "State serial: %s\n", st.Metadata.Serial)

			printBlobs := func(name string, Type resources.Type) {
				for snapshot, err := range st.ListObjectsOfType(Type) {
					if err != nil {
						fmt.Fprintf(ctx.Stdout, "Could not fetch blob entry for %s\n", name)
					} else {
						fmt.Fprintf(ctx.Stdout, "%s %x : packfile %x, offset %d, length %d\n",
							name,
							snapshot.Blob,
							snapshot.Location.Packfile,
							snapshot.Location.Offset,
							snapshot.Location.Length)
					}
				}
			}
			printDeleted := func(name string, Type resources.Type) {
				for deletedEntry, err := range st.ListDeletedResources(Type) {
					if err != nil {
						fmt.Fprintf(ctx.Stdout, "Could not fetch deleted blob entry for %s\n", name)
					} else {
						fmt.Fprintf(ctx.Stdout, "deleted %s: %x, when=%s\n",
							name,
							deletedEntry.Blob,
							deletedEntry.When)
					}
				}
			}

			for _, Type := range resources.Types() {
				printDeleted(Type.String(), Type)
				printBlobs(Type.String(), Type)
			}
		}
	}
	return 0, nil
}
