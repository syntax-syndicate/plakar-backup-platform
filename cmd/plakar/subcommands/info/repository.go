package info

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/dustin/go-humanize"
)

type InfoRepository struct {
	subcommands.SubcommandBase
	RepositorySecret []byte
}

func (cmd *InfoRepository) Parse(ctx *appcontext.AppContext, args []string) error {
	// Since this is the default action, we plug the general USAGE here.
	flags := flag.NewFlagSet("info", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s snapshot SNAPSHOT\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s errors SNAPSHOT\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s state [STATE]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s search snapshot[:path] mime\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s packfile [PACKFILE]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s object [OBJECT]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s vfs SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s xattr SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s contenttype SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s locks\n", flags.Name())
	}
	flags.Parse(args)

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *InfoRepository) Name() string {
	return "info_repository"
}

func (cmd *InfoRepository) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	fmt.Fprintln(ctx.Stdout, "Version:", repo.Configuration().Version)
	fmt.Fprintln(ctx.Stdout, "Timestamp:", repo.Configuration().Timestamp)
	fmt.Fprintln(ctx.Stdout, "RepositoryID:", repo.Configuration().RepositoryID)

	fmt.Fprintln(ctx.Stdout, "Packfile:")
	fmt.Fprintf(ctx.Stdout, " - MaxSize: %s (%d bytes)\n",
		humanize.Bytes(uint64(repo.Configuration().Packfile.MaxSize)),
		repo.Configuration().Packfile.MaxSize)

	fmt.Fprintln(ctx.Stdout, "Chunking:")
	fmt.Fprintln(ctx.Stdout, " - Algorithm:", repo.Configuration().Chunking.Algorithm)
	fmt.Fprintf(ctx.Stdout, " - MinSize: %s (%d bytes)\n",
		humanize.Bytes(uint64(repo.Configuration().Chunking.MinSize)), repo.Configuration().Chunking.MinSize)
	fmt.Fprintf(ctx.Stdout, " - NormalSize: %s (%d bytes)\n",
		humanize.Bytes(uint64(repo.Configuration().Chunking.NormalSize)), repo.Configuration().Chunking.NormalSize)
	fmt.Fprintf(ctx.Stdout, " - MaxSize: %s (%d bytes)\n",
		humanize.Bytes(uint64(repo.Configuration().Chunking.MaxSize)), repo.Configuration().Chunking.MaxSize)

	fmt.Fprintln(ctx.Stdout, "Hashing:")
	fmt.Fprintln(ctx.Stdout, " - Algorithm:", repo.Configuration().Hashing.Algorithm)
	fmt.Fprintln(ctx.Stdout, " - Bits:", repo.Configuration().Hashing.Bits)

	if repo.Configuration().Compression != nil {
		fmt.Fprintln(ctx.Stdout, "Compression:")
		fmt.Fprintln(ctx.Stdout, " - Algorithm:", repo.Configuration().Compression.Algorithm)
		fmt.Fprintln(ctx.Stdout, " - Level:", repo.Configuration().Compression.Level)
	}

	if repo.Configuration().Encryption != nil {
		fmt.Fprintln(ctx.Stdout, "Encryption:")
		fmt.Fprintln(ctx.Stdout, " - SubkeyAlgorithm:", repo.Configuration().Encryption.SubKeyAlgorithm)
		fmt.Fprintln(ctx.Stdout, " - DataAlgorithm:", repo.Configuration().Encryption.DataAlgorithm)
		fmt.Fprintln(ctx.Stdout, " - ChunkSize:", repo.Configuration().Encryption.ChunkSize)
		fmt.Fprintf(ctx.Stdout, " - Canary: %x\n", repo.Configuration().Encryption.Canary)
		fmt.Fprintln(ctx.Stdout, " - KDF:", repo.Configuration().Encryption.KDFParams.KDF)
		fmt.Fprintf(ctx.Stdout, "   - Salt: %x\n", repo.Configuration().Encryption.KDFParams.Salt)
		switch repo.Configuration().Encryption.KDFParams.KDF {
		case "ARGON2ID":
			fmt.Fprintf(ctx.Stdout, "   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.SaltSize)
			fmt.Fprintf(ctx.Stdout, "   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.KeyLen)
			fmt.Fprintf(ctx.Stdout, "   - Time: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Time)
			fmt.Fprintf(ctx.Stdout, "   - Memory: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Memory)
			fmt.Fprintf(ctx.Stdout, "   - Thread: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Threads)
		case "SCRYPT":
			fmt.Fprintf(ctx.Stdout, "   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.SaltSize)
			fmt.Fprintf(ctx.Stdout, "   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.KeyLen)
			fmt.Fprintf(ctx.Stdout, "   - N: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.N)
			fmt.Fprintf(ctx.Stdout, "   - R: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.R)
			fmt.Fprintf(ctx.Stdout, "   - P: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.P)
		case "PBKDF2":
			fmt.Fprintf(ctx.Stdout, "   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.SaltSize)
			fmt.Fprintf(ctx.Stdout, "   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.KeyLen)
			fmt.Fprintf(ctx.Stdout, "   - Iterations: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.Iterations)
			fmt.Fprintf(ctx.Stdout, "   - Hashing: %s\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.Hashing)
		default:
			fmt.Fprintf(ctx.Stdout, "   - Unsupported KDF: %s\n", repo.Configuration().Encryption.KDFParams.KDF)
		}
	}

	nSnapshots, logicalSize, err := snapshot.LogicalSize(repo)
	if err != nil {
		return 1, fmt.Errorf("unable to calculate logical size: %w", err)
	}

	fmt.Fprintln(ctx.Stdout, "Snapshots:", nSnapshots)

	storageSize := repo.Store().Size()

	fmt.Fprintf(ctx.Stdout, "Storage size: %s (%d bytes)\n", humanize.Bytes(uint64(storageSize)), uint64(storageSize))
	fmt.Fprintf(ctx.Stdout, "Logical size: %s (%d bytes)\n", humanize.Bytes(uint64(logicalSize)), logicalSize)

	efficiency := float64(0)
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
	fmt.Fprintf(ctx.Stdout, "Storage efficiency: %.2f%%\n", efficiency)

	return 0, nil
}
