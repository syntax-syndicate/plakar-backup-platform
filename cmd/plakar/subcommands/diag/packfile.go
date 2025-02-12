package info

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/repository"
)

type InfoPackfile struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Args []string
}

func (cmd *InfoPackfile) Name() string {
	return "info_packfile"
}

func (cmd *InfoPackfile) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.Args) == 0 {
		packfiles, err := repo.GetPackfiles()
		if err != nil {
			return 1, err
		}

		for _, packfile := range packfiles {
			fmt.Fprintf(ctx.Stdout, "%x\n", packfile)
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

			packfileVersion, rd, err := repo.GetPackfile(byteArray)
			if err != nil {
				return 1, err
			}

			rawPackfile, err := io.ReadAll(rd)
			if err != nil {
				return 1, err
			}

			footerBufLength := binary.LittleEndian.Uint32(rawPackfile[len(rawPackfile)-4:])
			rawPackfile = rawPackfile[:len(rawPackfile)-4]

			footerbuf := rawPackfile[len(rawPackfile)-int(footerBufLength):]
			rawPackfile = rawPackfile[:len(rawPackfile)-int(footerBufLength)]

			//fmt.Println(footerbuf, len(footerbuf))

			footerbuf, err = repo.DecodeBuffer(footerbuf)
			if err != nil {
				return 1, err
			}

			footer, err := packfile.NewFooterFromBytes(packfileVersion, footerbuf)
			if err != nil {
				return 1, err
			}

			indexbuf := rawPackfile[int(footer.IndexOffset):]
			rawPackfile = rawPackfile[:int(footer.IndexOffset)]

			indexbuf, err = repo.DecodeBuffer(indexbuf)
			if err != nil {
				return 1, err
			}

			hasher := repo.GetMACHasher()
			hasher.Write(indexbuf)

			if !bytes.Equal(hasher.Sum(nil), footer.IndexMAC[:]) {
				return 1, fmt.Errorf("index MAC mismatch")
			}

			rawPackfile = append(rawPackfile, indexbuf...)
			rawPackfile = append(rawPackfile, footerbuf...)

			p, err := packfile.NewFromBytes(repo.GetMACHasher(), packfileVersion, rawPackfile)
			if err != nil {
				return 1, err
			}

			fmt.Fprintf(ctx.Stdout, "Version: %s\n", p.Footer.Version)
			fmt.Fprintf(ctx.Stdout, "Timestamp: %s\n", time.Unix(0, p.Footer.Timestamp))
			fmt.Fprintf(ctx.Stdout, "Index MAC: %x\n", p.Footer.IndexMAC)
			fmt.Fprintln(ctx.Stdout)

			for i, entry := range p.Index {
				fmt.Fprintf(ctx.Stdout, "blob[%d]: %x %d %d %x %s\n", i, entry.MAC, entry.Offset, entry.Length, entry.Flags, entry.Type)
			}
		}
	}
	return 0, nil
}
