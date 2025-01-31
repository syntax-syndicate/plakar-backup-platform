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
	fmt.Printf("Version: %s\n", repo.Configuration().Version)
	fmt.Printf("SnapshotID: %s\n", hex.EncodeToString(indexID[:]))
	fmt.Printf("Timestamp: %s\n", header.Timestamp)
	fmt.Printf("Duration: %s\n", header.Duration)

	fmt.Printf("Name: %s\n", header.Name)
	fmt.Printf("Environment: %s\n", header.Environment)
	fmt.Printf("Perimeter: %s\n", header.Perimeter)
	fmt.Printf("Category: %s\n", header.Category)
	if len(header.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(header.Tags, ", "))
	}

	if header.Identity.Identifier != uuid.Nil {
		fmt.Println("Identity:")
		fmt.Printf(" - Identifier: %s\n", header.Identity.Identifier)
		fmt.Printf(" - PublicKey: %s\n", base64.RawStdEncoding.EncodeToString(header.Identity.PublicKey))
	}

	fmt.Printf("Root: %x\n", header.Root)
	fmt.Printf("Index: %x\n", header.Index)
	fmt.Printf("Metadata: %x\n", header.Metadata)
	fmt.Printf("Statistics: %x\n", header.Statistics)

	fmt.Println("Importer:")
	fmt.Printf(" - Type: %s\n", header.Importer.Type)
	fmt.Printf(" - Origin: %s\n", header.Importer.Origin)
	fmt.Printf(" - Directory: %s\n", header.Importer.Directory)

	fmt.Println("Context:")
	fmt.Printf(" - MachineID: %s\n", header.GetContext("MachineID"))
	fmt.Printf(" - Hostname: %s\n", header.GetContext("Hostname"))
	fmt.Printf(" - Username: %s\n", header.GetContext("Username"))
	fmt.Printf(" - OperatingSystem: %s\n", header.GetContext("OperatingSystem"))
	fmt.Printf(" - Architecture: %s\n", header.GetContext("Architecture"))
	fmt.Printf(" - NumCPU: %s\n", header.GetContext("NumCPU"))
	fmt.Printf(" - GOMAXPROCS: %s\n", header.GetContext("GOMAXPROCS"))
	fmt.Printf(" - ProcessID: %s\n", header.GetContext("ProcessID"))
	fmt.Printf(" - Client: %s\n", header.GetContext("Client"))
	fmt.Printf(" - CommandLine: %s\n", header.GetContext("CommandLine"))

	fmt.Println("Summary:")
	fmt.Printf(" - Directories: %d\n", header.Summary.Directory.Directories+header.Summary.Below.Directories)
	fmt.Printf(" - Files: %d\n", header.Summary.Directory.Files+header.Summary.Below.Files)
	fmt.Printf(" - Symlinks: %d\n", header.Summary.Directory.Symlinks+header.Summary.Below.Symlinks)
	fmt.Printf(" - Devices: %d\n", header.Summary.Directory.Devices+header.Summary.Below.Devices)
	fmt.Printf(" - Pipes: %d\n", header.Summary.Directory.Pipes+header.Summary.Below.Pipes)
	fmt.Printf(" - Sockets: %d\n", header.Summary.Directory.Sockets+header.Summary.Below.Sockets)
	fmt.Printf(" - Setuid: %d\n", header.Summary.Directory.Setuid+header.Summary.Below.Setuid)
	fmt.Printf(" - Setgid: %d\n", header.Summary.Directory.Setgid+header.Summary.Below.Setgid)
	fmt.Printf(" - Sticky: %d\n", header.Summary.Directory.Sticky+header.Summary.Below.Sticky)

	fmt.Printf(" - Objects: %d\n", header.Summary.Directory.Objects+header.Summary.Below.Objects)
	fmt.Printf(" - Chunks: %d\n", header.Summary.Directory.Chunks+header.Summary.Below.Chunks)
	fmt.Printf(" - MinSize: %s (%d bytes)\n", humanize.Bytes(min(header.Summary.Directory.MinSize, header.Summary.Below.MinSize)), min(header.Summary.Directory.MinSize, header.Summary.Below.MinSize))
	fmt.Printf(" - MaxSize: %s (%d bytes)\n", humanize.Bytes(max(header.Summary.Directory.MaxSize, header.Summary.Below.MaxSize)), max(header.Summary.Directory.MaxSize, header.Summary.Below.MaxSize))
	fmt.Printf(" - Size: %s (%d bytes)\n", humanize.Bytes(header.Summary.Directory.Size+header.Summary.Below.Size), header.Summary.Directory.Size+header.Summary.Below.Size)
	fmt.Printf(" - MinModTime: %s\n", time.Unix(min(header.Summary.Directory.MinModTime, header.Summary.Below.MinModTime), 0))
	fmt.Printf(" - MaxModTime: %s\n", time.Unix(max(header.Summary.Directory.MaxModTime, header.Summary.Below.MaxModTime), 0))
	fmt.Printf(" - MinEntropy: %f\n", min(header.Summary.Directory.MinEntropy, header.Summary.Below.MinEntropy))
	fmt.Printf(" - MaxEntropy: %f\n", max(header.Summary.Directory.MaxEntropy, header.Summary.Below.MaxEntropy))
	fmt.Printf(" - HiEntropy: %d\n", header.Summary.Directory.HiEntropy+header.Summary.Below.HiEntropy)
	fmt.Printf(" - LoEntropy: %d\n", header.Summary.Directory.LoEntropy+header.Summary.Below.LoEntropy)
	fmt.Printf(" - MIMEAudio: %d\n", header.Summary.Directory.MIMEAudio+header.Summary.Below.MIMEAudio)
	fmt.Printf(" - MIMEVideo: %d\n", header.Summary.Directory.MIMEVideo+header.Summary.Below.MIMEVideo)
	fmt.Printf(" - MIMEImage: %d\n", header.Summary.Directory.MIMEImage+header.Summary.Below.MIMEImage)
	fmt.Printf(" - MIMEText: %d\n", header.Summary.Directory.MIMEText+header.Summary.Below.MIMEText)
	fmt.Printf(" - MIMEApplication: %d\n", header.Summary.Directory.MIMEApplication+header.Summary.Below.MIMEApplication)
	fmt.Printf(" - MIMEOther: %d\n", header.Summary.Directory.MIMEOther+header.Summary.Below.MIMEOther)

	fmt.Printf(" - Errors: %d\n", header.Summary.Directory.Errors+header.Summary.Below.Errors)
	return 0, nil
}
