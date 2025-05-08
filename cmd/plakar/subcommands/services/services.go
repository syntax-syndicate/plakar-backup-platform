/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
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

package services

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Services{} }, subcommands.AgentSupport, "services")
}

func (cmd *Services) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("services", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() != 2 {
		flags.Usage()
		return fmt.Errorf("invalid number of arguments")
	}

	action := flags.Arg(0)
	parameter := flags.Arg(1)

	if action != "enable" && action != "disable" && action != "status" {
		flags.Usage()
		return fmt.Errorf("invalid action: %s, should be enable, disable, or status", action)
	}

	cmd.action = action
	cmd.parameter = parameter
	return nil
}

type Services struct {
	subcommands.SubcommandBase

	action    string
	parameter string
}

func (cmd *Services) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if cache, err := ctx.GetCache().Repository(repo.Configuration().RepositoryID); err != nil {
		fmt.Fprintf(ctx.Stdout, "####1\n")
		return 1, err
	} else if authToken, err := cache.GetAuthToken(); err != nil {
		fmt.Fprintf(ctx.Stdout, "####2\n")
		return 1, err
	} else if authToken == "" {
		fmt.Fprintf(ctx.Stdout, "####3\n")
		return 1, fmt.Errorf("access to services requires login, please run `plakar login`")
	} else {
		switch cmd.action {
		case "status":
			url := fmt.Sprintf("https://api.plakar.io/v1/account/services/%s", cmd.parameter)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return 1, fmt.Errorf("failed to create request: %v", err)
			}
			req.Header.Set("User-Agent", fmt.Sprintf("%s (%s/%s)", ctx.Client, ctx.OperatingSystem, ctx.Architecture))
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+authToken)
			req.Header.Set("Accept-Encoding", "gzip")
			req.Header.Set("Accept-Charset", "utf-8")

			httpClient := http.DefaultClient
			resp, err := httpClient.Do(req)
			if err != nil {
				return 1, fmt.Errorf("failed to get service status: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				return 1, fmt.Errorf("failed to get service status: %s", resp.Status)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return 1, fmt.Errorf("failed to read response body: %v", err)
			}

			var response struct {
				Enabled     bool `json:"enabled"`
				EmailReport bool `json:"email_report"`
			}
			if err := json.Unmarshal(data, &response); err != nil {
				return 1, fmt.Errorf("failed to unmarshal response: %v", err)
			}
			fmt.Fprintf(ctx.Stdout, "Service %s status:\n", cmd.parameter)
			fmt.Fprintf(ctx.Stdout, "  Enabled: %t\n", response.Enabled)
			fmt.Fprintf(ctx.Stdout, "  Email Report: %t\n", response.EmailReport)

			// Handle the response here
		}
		return 0, nil
	}
}
