package info

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

type InfoSnapshot struct {
	RepositoryLocation string
	RepositorySecret   []byte

	SnapshotID string
}

func (cmd *InfoSnapshot) Name() string {
	return "info_snapshot"
}

func (cmd *InfoSnapshot) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, err := utils.OpenSnapshotByPrefix(repo, cmd.SnapshotID)
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

	fmt.Fprintf(ctx.Stdout, "Root: %x\n", header.Root)
	fmt.Fprintf(ctx.Stdout, "Index: %x\n", header.Index)
	fmt.Fprintf(ctx.Stdout, "Metadata: %x\n", header.Metadata)
	fmt.Fprintf(ctx.Stdout, "Statistics: %x\n", header.Statistics)

	fmt.Fprintln(ctx.Stdout, "Importer:")
	fmt.Fprintf(ctx.Stdout, " - Type: %s\n", header.Importer.Type)
	fmt.Fprintf(ctx.Stdout, " - Origin: %s\n", header.Importer.Origin)
	fmt.Fprintf(ctx.Stdout, " - Directory: %s\n", header.Importer.Directory)

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

	fmt.Fprintln(ctx.Stdout, "Summary:")
	fmt.Fprintf(ctx.Stdout, " - Directories: %d\n", header.Summary.Directory.Directories+header.Summary.Below.Directories)
	fmt.Fprintf(ctx.Stdout, " - Files: %d\n", header.Summary.Directory.Files+header.Summary.Below.Files)
	fmt.Fprintf(ctx.Stdout, " - Symlinks: %d\n", header.Summary.Directory.Symlinks+header.Summary.Below.Symlinks)
	fmt.Fprintf(ctx.Stdout, " - Devices: %d\n", header.Summary.Directory.Devices+header.Summary.Below.Devices)
	fmt.Fprintf(ctx.Stdout, " - Pipes: %d\n", header.Summary.Directory.Pipes+header.Summary.Below.Pipes)
	fmt.Fprintf(ctx.Stdout, " - Sockets: %d\n", header.Summary.Directory.Sockets+header.Summary.Below.Sockets)
	fmt.Fprintf(ctx.Stdout, " - Setuid: %d\n", header.Summary.Directory.Setuid+header.Summary.Below.Setuid)
	fmt.Fprintf(ctx.Stdout, " - Setgid: %d\n", header.Summary.Directory.Setgid+header.Summary.Below.Setgid)
	fmt.Fprintf(ctx.Stdout, " - Sticky: %d\n", header.Summary.Directory.Sticky+header.Summary.Below.Sticky)

	fmt.Fprintf(ctx.Stdout, " - Objects: %d\n", header.Summary.Directory.Objects+header.Summary.Below.Objects)
	fmt.Fprintf(ctx.Stdout, " - Chunks: %d\n", header.Summary.Directory.Chunks+header.Summary.Below.Chunks)
	fmt.Fprintf(ctx.Stdout, " - MinSize: %s (%d bytes)\n", humanize.Bytes(min(header.Summary.Directory.MinSize, header.Summary.Below.MinSize)), min(header.Summary.Directory.MinSize, header.Summary.Below.MinSize))
	fmt.Fprintf(ctx.Stdout, " - MaxSize: %s (%d bytes)\n", humanize.Bytes(max(header.Summary.Directory.MaxSize, header.Summary.Below.MaxSize)), max(header.Summary.Directory.MaxSize, header.Summary.Below.MaxSize))
	fmt.Fprintf(ctx.Stdout, " - Size: %s (%d bytes)\n", humanize.Bytes(header.Summary.Directory.Size+header.Summary.Below.Size), header.Summary.Directory.Size+header.Summary.Below.Size)
	fmt.Fprintf(ctx.Stdout, " - MinModTime: %s\n", time.Unix(min(header.Summary.Directory.MinModTime, header.Summary.Below.MinModTime), 0))
	fmt.Fprintf(ctx.Stdout, " - MaxModTime: %s\n", time.Unix(max(header.Summary.Directory.MaxModTime, header.Summary.Below.MaxModTime), 0))
	fmt.Fprintf(ctx.Stdout, " - MinEntropy: %f\n", min(header.Summary.Directory.MinEntropy, header.Summary.Below.MinEntropy))
	fmt.Fprintf(ctx.Stdout, " - MaxEntropy: %f\n", max(header.Summary.Directory.MaxEntropy, header.Summary.Below.MaxEntropy))
	fmt.Fprintf(ctx.Stdout, " - HiEntropy: %d\n", header.Summary.Directory.HiEntropy+header.Summary.Below.HiEntropy)
	fmt.Fprintf(ctx.Stdout, " - LoEntropy: %d\n", header.Summary.Directory.LoEntropy+header.Summary.Below.LoEntropy)
	fmt.Fprintf(ctx.Stdout, " - MIMEAudio: %d\n", header.Summary.Directory.MIMEAudio+header.Summary.Below.MIMEAudio)
	fmt.Fprintf(ctx.Stdout, " - MIMEVideo: %d\n", header.Summary.Directory.MIMEVideo+header.Summary.Below.MIMEVideo)
	fmt.Fprintf(ctx.Stdout, " - MIMEImage: %d\n", header.Summary.Directory.MIMEImage+header.Summary.Below.MIMEImage)
	fmt.Fprintf(ctx.Stdout, " - MIMEText: %d\n", header.Summary.Directory.MIMEText+header.Summary.Below.MIMEText)
	fmt.Fprintf(ctx.Stdout, " - MIMEApplication: %d\n", header.Summary.Directory.MIMEApplication+header.Summary.Below.MIMEApplication)
	fmt.Fprintf(ctx.Stdout, " - MIMEOther: %d\n", header.Summary.Directory.MIMEOther+header.Summary.Below.MIMEOther)

	fmt.Fprintf(ctx.Stdout, " - Errors: %d\n", header.Summary.Directory.Errors+header.Summary.Below.Errors)
	return 0, nil
}
