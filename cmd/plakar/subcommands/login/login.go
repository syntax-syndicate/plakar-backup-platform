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
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
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

	flow, err := NewLoginFlow()
	if err != nil {
		return 1, err
	}

	defer flow.Close()

	var token string

	if cmd.Github {
		token, err = flow.RunGithub(!cmd.NoSpawn)
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

type loginFlow struct {
	server  *http.Server
	port    int
	channel chan string
}

// NewLoginFlow spawns a local HTTP server in a goroutine, listening for the
// OAuth callback on a random port.
func NewLoginFlow() (*loginFlow, error) {
	flow := &loginFlow{}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen on a random port: %w", err)
	}
	flow.port = listener.Addr().(*net.TCPAddr).Port

	flow.channel = make(chan string)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")

		if token != "" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(loginOkHTML))
			flow.channel <- token
		} else {
			fmt.Fprintf(w, "Waiting for OAuth callback with token...")
		}
	})

	flow.server = &http.Server{Handler: mux}
	go func() {
		if err := flow.server.Serve(listener); err != nil {
			if err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}
	}()
	return flow, nil
}

//go:embed loginOk.html
var loginOkHTML string

// RunGithub sends a POST request to the login endpoint and waits for the user to
// complete the OAuth flow.
func (flow *loginFlow) RunGithub(spawnBrowser bool) (string, error) {
	reqBody := map[string]string{
		"mode":     "headers",
		"redirect": fmt.Sprintf("http://localhost:%d", flow.port),
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
		URL string `json:"URL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	if spawnBrowser {
		switch runtime.GOOS {
		case "windows":
			err = exec.Command("rundll32", "url.dll,FileProtocolHandler", respData.URL).Start()
		case "darwin":
			err = exec.Command("open", respData.URL).Start()
		default: // "linux", "freebsd", "openbsd", "netbsd"
			err = exec.Command("xdg-open", respData.URL).Start()
		}
	}
	if !spawnBrowser || err != nil {
		fmt.Printf("Open the following URL in your browser to login:\n\n%s\n", respData.URL)
	}

	// Wait for the token to be received
	fmt.Printf("Waiting for your browser to complete the login...\n")
	token := <-flow.channel

	return token, nil
}

func (flow *loginFlow) RunEmail(email string) (string, error) {
	reqBody := map[string]string{
		"email":    email,
		"mode":     "headers",
		"redirect": fmt.Sprintf("http://localhost:%d", flow.port),
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
	token := <-flow.channel

	return token, nil
}

func (flow *loginFlow) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := flow.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %v", err)
	}
	return nil
}
