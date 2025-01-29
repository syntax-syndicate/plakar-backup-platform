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

package agent

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register("agent", cmd_agent)
}

func cmd_agent(ctx *appcontext.AppContext, _ *repository.Repository, args []string) (int, error) {
	var opt_prometheus string
	var opt_socketPath string

	flags := flag.NewFlagSet("agent", flag.ExitOnError)
	flags.StringVar(&opt_prometheus, "prometheus", "", "prometheus exporter interface")
	flags.StringVar(&opt_socketPath, "socket", filepath.Join(ctx.CacheDir, "agent.sock"), "path to socket file")
	flags.Parse(args)

	daemon, err := agent.NewAgent(ctx, "unix", opt_socketPath, opt_prometheus)
	if err != nil {
		ctx.GetLogger().Error("failed to create agent daemon: %s", err)
		return 1, err
	}
	defer daemon.Close()

	if err := daemon.ListenAndServe(handleRPC); err != nil {
		return 1, err
	}

	log.Println("Server gracefully stopped")

	return 0, nil
}

func handleRPC(clientContext *appcontext.AppContext, repo *repository.Repository, command string, args []string) (int, error) {
	return subcommands.Execute(clientContext, repo, command, args, true)
}
