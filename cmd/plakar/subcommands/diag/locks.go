package diag

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
)

type DiagLocks struct {
	RepositorySecret []byte
}

func (cmd *DiagLocks) Name() string {
	return "diag_locks"
}

func (cmd *DiagLocks) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	locksID, err := repo.GetLocks()
	if err != nil {
		return 1, err
	}

	for _, lockID := range locksID {
		version, rd, err := repo.GetLock(lockID)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to fetch lock %x\n", lockID)
		}

		lock, err := repository.NewLockFromStream(version, rd)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to deserialize lock %x\n", lockID)
		}

		var lockType string
		if lock.Exclusive {
			lockType = "exclusive"
		} else {
			lockType = "shared"
		}

		fmt.Fprintf(ctx.Stdout, "[%x] Got %s access on %s owner %s\n", lockID, lockType, lock.Timestamp.UTC().Format(time.RFC3339), lock.Hostname)
	}

	return 0, nil
}
