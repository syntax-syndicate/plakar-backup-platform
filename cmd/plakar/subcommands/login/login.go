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

package login

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Login{} }, 0, "login")
}

func (cmd *Login) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_nospawn bool
	var opt_github bool
	var opt_email string

	flags := flag.NewFlagSet("login", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&opt_nospawn, "no-spawn", false, "don't spawn browser")
	flags.BoolVar(&opt_github, "github", false, "login with GitHub")
	flags.StringVar(&opt_email, "email", "", "login with email")
	flags.Parse(args)

	if opt_github && opt_email != "" {
		return fmt.Errorf("specify either -github or -email, not both")
	}

	if !opt_github && opt_email == "" {
		return fmt.Errorf("specify either -github or -email")
	}

	if opt_nospawn && !opt_github {
		return fmt.Errorf("the -no-spawn option is only valid with -github")
	}

	cmd.Github = opt_github
	cmd.Email = opt_email
	cmd.NoSpawn = opt_nospawn

	return nil
}

type Login struct {
	subcommands.SubcommandBase

	Github  bool
	Email   string
	NoSpawn bool
}

func (cmd *Login) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var err error

	flow, err := NewLoginFlow(ctx)
	if err != nil {
		return 1, err
	}

	defer flow.Close()

	var token string

	if cmd.Github {
		token, err = flow.RunGithub()
	} else if cmd.Email != "" {
		token, err = flow.RunEmail(cmd.Email)
	} else {
		return 1, fmt.Errorf("invalid login method")
	}
	if err != nil {
		return 1, err
	}

	if cache, err := ctx.GetCache().Repository(repo.Configuration().RepositoryID); err != nil {
		return 1, fmt.Errorf("failed to get repository cache: %w", err)
	} else if err := cache.PutAuthToken(token); err != nil {
		return 1, fmt.Errorf("failed to store token in cache: %w", err)
	}

	return 0, nil
}

type TokenResponse struct {
	Token string `json:"token"`
}

type loginFlow struct {
	appCtx *appcontext.AppContext
}

// NewLoginFlow spawns a local HTTP server in a goroutine, listening for the
// OAuth callback on a random port.
func NewLoginFlow(appCtx *appcontext.AppContext) (*loginFlow, error) {
	flow := &loginFlow{
		appCtx: appCtx,
	}
	return flow, nil
}

// RunGithub sends a POST request to the login endpoint and waits for the user to
// complete the OAuth flow.
func (flow *loginFlow) RunGithub() (string, error) {
	reqBody := map[string]string{
		"mode": "headers",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// XXX: make backend URL configurable

	resp, err := http.Post("http://localhost:8080/v1/auth/login/github", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("unable to get the login URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var respData struct {
		URL    string `json:"URL"`
		PollID string `json:"poll_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	fmt.Printf("\nPlease open the following URL in your browser:\n\n")
	fmt.Printf("  %s\n\n", respData.URL)

	// Wait for the token to be received
	fmt.Printf("Waiting for your browser to complete the login...\n")

	for _ = range 5 {
		tick := time.After(1 * time.Second)
		select {
		case <-flow.appCtx.Done():
			return "", flow.appCtx.Err()
		case <-tick:
			reqUrl := "http://localhost:8080/v1/auth/poll/" + respData.PollID
			req, err := http.NewRequestWithContext(flow.appCtx.Context, "POST", reqUrl, nil)
			if err != nil {
				return "", fmt.Errorf("the /auth/login/github/poll API endpoint failed: %w", err)
			}

			client := http.DefaultClient
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("the /auth/login/github/poll API endpoint failed: %w", err)
			}
			// leaking resp for now
			if resp.StatusCode == http.StatusOK {
				var tokenResponse TokenResponse
				if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
					return "", fmt.Errorf("failed to decode response JSON: %v", err)
				}
				return tokenResponse.Token, nil
			} else if resp.StatusCode == http.StatusNotFound {
				return "", fmt.Errorf("unknown ID")
			} else if resp.StatusCode == http.StatusAccepted {
				fmt.Print(".")
			} else {
				return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}
		}
	}

	return "", fmt.Errorf("login flow timed out")
}

func (flow *loginFlow) RunEmail(email string) (string, error) {
	reqBody := map[string]string{
		"email": email,
		"mode":  "headers",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// XXX: make backend URL configurable
	resp, err := http.Post("http://localhost:8080/v1/auth/login/email", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("the /auth/login/email API endpoint failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	fmt.Printf("\nCheck your email for the login link. Do not close this window until you have logged in.\n")
	//	token := <-flow.channel

	return "", nil
}

func (flow *loginFlow) Close() error {
	return nil
}
