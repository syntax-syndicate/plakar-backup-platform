package info

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

type InfoSnapshot struct {
	subcommands.SubcommandBase

	SnapshotID string
}

func (cmd *InfoSnapshot) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("info snapshot", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s snapshot SNAPSHOT", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotID = flags.Args()[0]

	return nil
}

func (cmd *InfoSnapshot) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, _, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotID)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	header := snap.Header

	indexID := header.GetIndexID()
	fmt.Fprintf(ctx.Stdout, "Version: %s\n", repo.Configuration().Version)
	fmt.Fprintf(ctx.Stdout, "SnapshotID: %s\n", hex.EncodeToString(indexID[:]))
	fmt.Fprintf(ctx.Stdout, "Timestamp: %s\n", header.Timestamp)
	fmt.Fprintf(ctx.Stdout, "Duration: %s\n", header.Duration)

	fmt.Fprintf(ctx.Stdout, "Name: %s\n", header.Name)
	fmt.Fprintf(ctx.Stdout, "Environment: %s\n", header.Environment)
	fmt.Fprintf(ctx.Stdout, "Perimeter: %s\n", header.Perimeter)
	fmt.Fprintf(ctx.Stdout, "Category: %s\n", header.Category)
	if len(header.Tags) > 0 {
		fmt.Fprintf(ctx.Stdout, "Tags: %s\n", strings.Join(header.Tags, ", "))
	}

	if header.Identity.Identifier != uuid.Nil {
		fmt.Fprintln(ctx.Stdout, "Identity:")
		fmt.Fprintf(ctx.Stdout, " - Identifier: %s\n", header.Identity.Identifier)
		fmt.Fprintf(ctx.Stdout, " - PublicKey: %s\n", base64.RawStdEncoding.EncodeToString(header.Identity.PublicKey))
	}

	fmt.Fprintln(ctx.Stdout, "Context:")
	fmt.Fprintf(ctx.Stdout, " - MachineID: %s\n", header.GetContext("MachineID"))
	fmt.Fprintf(ctx.Stdout, " - Hostname: %s\n", header.GetContext("Hostname"))
	fmt.Fprintf(ctx.Stdout, " - Username: %s\n", header.GetContext("Username"))
	fmt.Fprintf(ctx.Stdout, " - OperatingSystem: %s\n", header.GetContext("OperatingSystem"))
	fmt.Fprintf(ctx.Stdout, " - Architecture: %s\n", header.GetContext("Architecture"))
	fmt.Fprintf(ctx.Stdout, " - NumCPU: %s\n", header.GetContext("NumCPU"))
	fmt.Fprintf(ctx.Stdout, " - GOMAXPROCS: %s\n", header.GetContext("GOMAXPROCS"))
	fmt.Fprintf(ctx.Stdout, " - ProcessID: %s\n", header.GetContext("ProcessID"))
	fmt.Fprintf(ctx.Stdout, " - Client: %s\n", header.GetContext("Client"))
	fmt.Fprintf(ctx.Stdout, " - CommandLine: %s\n", header.GetContext("CommandLine"))

	for _, source := range snap.Header.Sources {
		fmt.Fprintf(ctx.Stdout, "VFS: %x\n", source.VFS)

		fmt.Fprintln(ctx.Stdout, "Importer:")
		fmt.Fprintf(ctx.Stdout, " - Type: %s\n", source.Importer.Type)
		fmt.Fprintf(ctx.Stdout, " - Origin: %s\n", source.Importer.Origin)
		fmt.Fprintf(ctx.Stdout, " - Directory: %s\n", source.Importer.Directory)

		fmt.Fprintln(ctx.Stdout, "Summary:")
		fmt.Fprintf(ctx.Stdout, " - Directories: %d\n", source.Summary.Directory.Directories+source.Summary.Below.Directories)
		fmt.Fprintf(ctx.Stdout, " - Files: %d\n", source.Summary.Directory.Files+source.Summary.Below.Files)
		fmt.Fprintf(ctx.Stdout, " - Symlinks: %d\n", source.Summary.Directory.Symlinks+source.Summary.Below.Symlinks)
		fmt.Fprintf(ctx.Stdout, " - Devices: %d\n", source.Summary.Directory.Devices+source.Summary.Below.Devices)
		fmt.Fprintf(ctx.Stdout, " - Pipes: %d\n", source.Summary.Directory.Pipes+source.Summary.Below.Pipes)
		fmt.Fprintf(ctx.Stdout, " - Sockets: %d\n", source.Summary.Directory.Sockets+source.Summary.Below.Sockets)
		fmt.Fprintf(ctx.Stdout, " - Setuid: %d\n", source.Summary.Directory.Setuid+source.Summary.Below.Setuid)
		fmt.Fprintf(ctx.Stdout, " - Setgid: %d\n", source.Summary.Directory.Setgid+source.Summary.Below.Setgid)
		fmt.Fprintf(ctx.Stdout, " - Sticky: %d\n", source.Summary.Directory.Sticky+source.Summary.Below.Sticky)

		fmt.Fprintf(ctx.Stdout, " - Objects: %d\n", source.Summary.Directory.Objects+source.Summary.Below.Objects)
		fmt.Fprintf(ctx.Stdout, " - Chunks: %d\n", source.Summary.Directory.Chunks+source.Summary.Below.Chunks)
		fmt.Fprintf(ctx.Stdout, " - MinSize: %s (%d bytes)\n", humanize.Bytes(min(source.Summary.Directory.MinSize, source.Summary.Below.MinSize)), min(source.Summary.Directory.MinSize, source.Summary.Below.MinSize))
		fmt.Fprintf(ctx.Stdout, " - MaxSize: %s (%d bytes)\n", humanize.Bytes(max(source.Summary.Directory.MaxSize, source.Summary.Below.MaxSize)), max(source.Summary.Directory.MaxSize, source.Summary.Below.MaxSize))
		fmt.Fprintf(ctx.Stdout, " - Size: %s (%d bytes)\n", humanize.Bytes(source.Summary.Directory.Size+source.Summary.Below.Size), source.Summary.Directory.Size+source.Summary.Below.Size)
		fmt.Fprintf(ctx.Stdout, " - MinModTime: %s\n", time.Unix(min(source.Summary.Directory.MinModTime, source.Summary.Below.MinModTime), 0))
		fmt.Fprintf(ctx.Stdout, " - MaxModTime: %s\n", time.Unix(max(source.Summary.Directory.MaxModTime, source.Summary.Below.MaxModTime), 0))
		fmt.Fprintf(ctx.Stdout, " - MinEntropy: %f\n", min(source.Summary.Directory.MinEntropy, source.Summary.Below.MinEntropy))
		fmt.Fprintf(ctx.Stdout, " - MaxEntropy: %f\n", max(source.Summary.Directory.MaxEntropy, source.Summary.Below.MaxEntropy))
		fmt.Fprintf(ctx.Stdout, " - HiEntropy: %d\n", source.Summary.Directory.HiEntropy+source.Summary.Below.HiEntropy)
		fmt.Fprintf(ctx.Stdout, " - LoEntropy: %d\n", source.Summary.Directory.LoEntropy+source.Summary.Below.LoEntropy)
		fmt.Fprintf(ctx.Stdout, " - MIMEAudio: %d\n", source.Summary.Directory.MIMEAudio+source.Summary.Below.MIMEAudio)
		fmt.Fprintf(ctx.Stdout, " - MIMEVideo: %d\n", source.Summary.Directory.MIMEVideo+source.Summary.Below.MIMEVideo)
		fmt.Fprintf(ctx.Stdout, " - MIMEImage: %d\n", source.Summary.Directory.MIMEImage+source.Summary.Below.MIMEImage)
		fmt.Fprintf(ctx.Stdout, " - MIMEText: %d\n", source.Summary.Directory.MIMEText+source.Summary.Below.MIMEText)
		fmt.Fprintf(ctx.Stdout, " - MIMEApplication: %d\n", source.Summary.Directory.MIMEApplication+source.Summary.Below.MIMEApplication)
		fmt.Fprintf(ctx.Stdout, " - MIMEOther: %d\n", source.Summary.Directory.MIMEOther+source.Summary.Below.MIMEOther)

		fmt.Fprintf(ctx.Stdout, " - Errors: %d\n", source.Summary.Directory.Errors+source.Summary.Below.Errors)

		fmt.Fprintf(ctx.Stdout, " ------------------------\n") // Separator for each source
	}

	return 0, nil
}
