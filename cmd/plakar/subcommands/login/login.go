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
	"io"
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
		token, err = flow.Run("github", map[string]string{})
	} else if cmd.Email != "" {
		token, err = flow.Run("email", map[string]string{"email": cmd.Email})
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

func (flow *loginFlow) Poll(pollID string, iterations int, delay time.Duration) (string, error) {
	for range iterations {
		tick := time.After(delay)
		select {
		case <-flow.appCtx.Done():
			return "", flow.appCtx.Err()
		case <-tick:
			reqUrl := "https://api.plakar.io/v1/auth/poll/" + pollID
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
	return "", fmt.Errorf("could not obtain token after %d iterations", iterations)
}

func (flow *loginFlow) Run(provider string, parameters map[string]string) (string, error) {
	var url string
	var body io.Reader

	switch provider {
	case "github":
		url = "https://api.plakar.io/v1/auth/login/github"
	case "email":
		url = "https://api.plakar.io/v1/auth/login/email"
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	if bodyBytes, err := json.Marshal(parameters); err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	} else {
		body = bytes.NewBuffer(bodyBytes)
	}

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		return "", fmt.Errorf("unable to get the login URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d : %s", resp.StatusCode, data)
	}

	switch provider {
	case "github":
		return flow.handleGithubResponse(resp)
	case "email":
		return flow.handleEmailResponse(resp)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (flow *loginFlow) handleGithubResponse(resp *http.Response) (string, error) {
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

	return flow.Poll(respData.PollID, 10, time.Second*5)
}

func (flow *loginFlow) handleEmailResponse(resp *http.Response) (string, error) {
	var respData struct {
		PollID string `json:"poll_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	fmt.Printf("\nCheck your email for the login link. Do not close this window until you have logged in.\n")

	return flow.Poll(respData.PollID, 10, time.Second*5)
}

func (flow *loginFlow) Close() error {
	return nil
}
