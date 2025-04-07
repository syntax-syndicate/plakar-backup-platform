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

package backup

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/dustin/go-humanize"
	"github.com/gobwas/glob"
)

func init() {
	subcommands.Register("backup", parse_cmd_backup)
}

type excludeFlags []string

func (e *excludeFlags) String() string {
	return strings.Join(*e, ",")
}

func (e *excludeFlags) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func parse_cmd_backup(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	var opt_tags string
	var opt_excludes string
	var opt_exclude excludeFlags
	var opt_concurrency uint64
	var opt_quiet bool
	var opt_silent bool
	var opt_check bool
	// var opt_stdio bool

	excludes := []string{}

	flags := flag.NewFlagSet("backup", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] path\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [OPTIONS] @LOCATION\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Uint64Var(&opt_concurrency, "concurrency", uint64(ctx.MaxConcurrency), "maximum number of parallel tasks")
	flags.StringVar(&opt_tags, "tag", "", "tag to assign to this snapshot")
	flags.StringVar(&opt_excludes, "excludes", "", "path to a file containing newline-separated regex patterns, treated as -exclude")
	flags.Var(&opt_exclude, "exclude", "glob pattern to exclude files, can be specified multiple times to add several exclusion patterns")
	flags.BoolVar(&opt_quiet, "quiet", false, "suppress output")
	flags.BoolVar(&opt_silent, "silent", false, "suppress ALL output")
	flags.BoolVar(&opt_check, "check", false, "check the snapshot after creating it")
	//flags.BoolVar(&opt_stdio, "stdio", false, "output one line per file to stdout instead of the default interactive output")
	flags.Parse(args)

	for _, item := range opt_exclude {
		if _, err := glob.Compile(item); err != nil {
			return nil, fmt.Errorf("failed to compile exclude pattern: %s", item)
		}
		excludes = append(excludes, item)
	}

	if opt_excludes != "" {
		fp, err := os.Open(opt_excludes)
		if err != nil {
			return nil, fmt.Errorf("unable to open excludes file: %w", err)
		}
		defer fp.Close()

		scanner := bufio.NewScanner(fp)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := glob.Compile(line)
			if err != nil {
				return nil, fmt.Errorf("failed to compile exclude pattern: %s", line)
			}
			excludes = append(excludes, line)
		}
		if err := scanner.Err(); err != nil {
			ctx.GetLogger().Error("%s", err)
			return nil, err
		}
	}
	return &Backup{
		RepositorySecret: ctx.GetSecret(),
		Concurrency:      opt_concurrency,
		Tags:             opt_tags,
		Excludes:         excludes,
		Quiet:            opt_quiet,
		Path:             flags.Arg(0),
		OptCheck:         opt_check,
	}, nil
}

type Backup struct {
	RepositorySecret []byte
	Job              string

	Concurrency uint64
	Tags        string
	Excludes    []string
	Silent      bool
	Quiet       bool
	Path        string
	OptCheck    bool
}

func (cmd *Backup) Name() string {
	return "backup"
}

func (cmd *Backup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, err := snapshot.New(repo)
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return 1, err
	}
	defer snap.Close()

	if cmd.Job != "" {
		snap.Header.Job = cmd.Job
	}

	var tags []string
	if cmd.Tags == "" {
		tags = []string{}
	} else {
		tags = []string{cmd.Tags}
	}

	excludes := []glob.Glob{}
	for _, item := range cmd.Excludes {
		g, err := glob.Compile(item)
		if err != nil {
			return 1, fmt.Errorf("failed to compile exclude pattern: %s", item)
		}
		excludes = append(excludes, g)
	}

	opts := &snapshot.BackupOptions{
		MaxConcurrency: cmd.Concurrency,
		Name:           "default",
		Tags:           tags,
		Excludes:       excludes,
	}

	scanDir := ctx.CWD
	if cmd.Path != "" {
		scanDir = cmd.Path
	}

	importerConfig := map[string]string{
		"location": scanDir,
	}
	if strings.HasPrefix(scanDir, "@") {
		remote, ok := ctx.Config.GetRemote(scanDir[1:])
		if !ok {
			return 1, fmt.Errorf("could not resolve importer: %s", scanDir)
		}
		if _, ok := remote["location"]; !ok {
			return 1, fmt.Errorf("could not resolve importer location: %s", scanDir)
		} else {
			importerConfig = remote
		}
	}

	imp, err := importer.NewImporter(importerConfig)
	if err != nil {
		if !filepath.IsAbs(scanDir) {
			scanDir = filepath.Join(ctx.CWD, scanDir)
		}
		imp, err = importer.NewImporter(map[string]string{"location": "fs://" + scanDir})
		if err != nil {
			return 1, fmt.Errorf("failed to create an importer for %s: %s", scanDir, err)
		}
	}
	defer imp.Close()

	if cmd.Silent {
		if err := snap.Backup(imp, opts); err != nil {
			return 1, fmt.Errorf("failed to create snapshot: %w", err)
		}
	} else {
		ep := startEventsProcessor(ctx, imp.Root(), true, cmd.Quiet)
		if err := snap.Backup(imp, opts); err != nil {
			ep.Close()
			return 1, fmt.Errorf("failed to create snapshot: %w", err)
		}
		ep.Close()
	}

	if cmd.OptCheck {
		repo.RebuildState()

		checkOptions := &snapshot.CheckOptions{
			MaxConcurrency: cmd.Concurrency,
			FastCheck:      false,
		}

		checkSnap, err := snapshot.Load(repo, snap.Header.Identifier)
		if err != nil {
			return 1, fmt.Errorf("failed to load snapshot: %w", err)
		}
		defer checkSnap.Close()

		ok, err := checkSnap.Check("/", checkOptions)
		if err != nil {
			return 1, fmt.Errorf("failed to check snapshot: %w", err)
		}
		if !ok {
			return 1, fmt.Errorf("snapshot is not valid")
		}
	}

	totalSize := snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size
	savings := float64(totalSize-uint64(snap.Repository().WBytes())) / float64(totalSize) * 100

	if uint64(snap.Repository().RBytes()) > totalSize {
		savings = 0
	}

	ctx.GetLogger().Info("%s: created %s snapshot %x of size %s in %s (wrote %s, saved %0.2f%%)",
		cmd.Name(),
		"unsigned",
		snap.Header.GetIndexShortID(),
		humanize.Bytes(totalSize),
		snap.Header.Duration,
		humanize.Bytes(uint64(snap.Repository().WBytes())),
		savings,
	)
	return 0, nil
}
