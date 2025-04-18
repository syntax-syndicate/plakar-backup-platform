/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package diag

import (
	"fmt"
	"iter"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
)

type DiagBlob struct {
	RepositorySecret []byte

	SnapshotPath string
	Slow bool
}

func (cmd *DiagBlob) Name() string {
	return "diag_blob"
}

func (cmd *DiagBlob) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, _, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, nil
	}
	defer snap.Close()

	var it iter.Seq2[resources.Type, objects.MAC]
	var iterErr error
	if cmd.Slow {
		it = snap.CrawlMACs(&iterErr)
	} else {
		it = snap.MACs(&iterErr)
	}

	for res, mac := range it {
		fmt.Fprintf(ctx.Stdout, "%s %x\n", res.String(), mac)
	}
	if iterErr != nil {
		return 1, iterErr
	}
	return 0, nil
}
