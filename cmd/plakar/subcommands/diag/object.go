package diag

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
)

type DiagObject struct {
	subcommands.SubcommandBase

	ObjectID string
}

func (cmd *DiagObject) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag objects", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s object OBJECT", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.ObjectID = flags.Args()[0]

	return nil
}

func (cmd *DiagObject) Name() string {
	return "diag_object"
}

func (cmd *DiagObject) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.ObjectID) != 64 {
		return 1, fmt.Errorf("invalid object hash: %s", cmd.ObjectID)
	}

	b, err := hex.DecodeString(cmd.ObjectID)
	if err != nil {
		return 1, fmt.Errorf("invalid object hash: %s", cmd.ObjectID)
	}

	// Convert the byte slice to a [32]byte
	var byteArray [32]byte
	copy(byteArray[:], b)

	rd, err := repo.GetBlob(resources.RT_OBJECT, byteArray)
	if err != nil {
		return 1, err
	}

	blob, err := io.ReadAll(rd)
	if err != nil {
		return 1, err
	}

	object, err := objects.NewObjectFromBytes(blob)
	if err != nil {
		return 1, err
	}

	fmt.Fprintf(ctx.Stdout, "object: %x\n", object.ContentMAC)
	fmt.Fprintln(ctx.Stdout, "  type:", object.ContentType)
	fmt.Fprintln(ctx.Stdout, "  chunks:")
	for _, chunk := range object.Chunks {
		fmt.Fprintf(ctx.Stdout, "    MAC: %x\n", chunk.ContentMAC)
	}
	return 0, nil
}
