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
	"path/filepath"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc"
	agent_handler "github.com/PlakarKorp/plakar/rpc/agent"
)

func init() {
	subcommands.Register2("agent", parse_cmd_agent)
}

func parse_cmd_agent(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (rpc.RPC, error) {
	var opt_prometheus string
	var opt_socketPath string

	flags := flag.NewFlagSet("agent", flag.ExitOnError)
	flags.StringVar(&opt_prometheus, "prometheus", "", "prometheus exporter interface")
	flags.StringVar(&opt_socketPath, "socket", filepath.Join(ctx.CacheDir, "agent.sock"), "path to socket file")
	flags.Parse(args)

	return &agent_handler.Agent{
		Prometheus: opt_prometheus,
		SocketPath: opt_socketPath,
	}, nil
}
