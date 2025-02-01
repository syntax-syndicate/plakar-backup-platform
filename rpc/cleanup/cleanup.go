package cleanup

import (
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
)

type Cleanup struct {
	RepositoryLocation string
	RepositorySecret   []byte
}

func (cmd *Cleanup) Name() string {
	return "cleanup"
}

func (cmd *Cleanup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	// the cleanup algorithm is a bit tricky and needs to be done in the correct sequence,
	// here's what it has to do:
	//
	// 1. fetch all snapshot indexes to figure out which blobs, objects and chunks are used
	// 2. blobs that are no longer in use can be be removed
	// 3. for each object and chunk, track which packfiles contain them
	// 4. if objects or chunks are present in more than one packfile...
	// 5. decide which one keeps it and a new packfile has to be generated for the other that contains everything BUT the object/chunk
	// 6. update indexes to reflect the new packfile
	// 7. save the new index
	return 0, nil
}
