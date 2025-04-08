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

package login

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"golang.org/x/oauth2"
)

func init() {
	subcommands.Register("login", parse_cmd_login)
}

func parse_cmd_login(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	var opt_provider string

	flags := flag.NewFlagSet("login", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}
	flags.StringVar(&opt_provider, "provider", "", "OAuth2 provider to use (github, google, etc.)")
	flags.Parse(args)

	if flags.NArg() != 1 {
		flags.Usage()
		return nil, fmt.Errorf("invalid arguments")
	}

	_, err := mail.ParseAddress(flags.Arg(0))
	if err != nil {
		return nil, fmt.Errorf("invalid email address: %s", flags.Arg(0))
	}

	return &Login{
		Email:    strings.ToLower(flags.Arg(0)),
		Provider: opt_provider,
	}, nil
}

type Login struct {
	//
	Email    string
	Provider string
}

type LoginRequest struct {
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

type LoginResponse struct {
	URL string `json:"url"`
}

func (cmd *Login) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	fmt.Println("DOING A LOGIN")

	//reader := bufio.NewReader(os.Stdin)
	//fmt.Print("Enter your email address: ")
	//email, _ := reader.ReadString('\n')
	//email = strings.TrimSpace(email)
	//	fmt.Println(email)

	req := LoginRequest{
		Email:    cmd.Email,
		Provider: cmd.Provider,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return 1, err
	}

	endpoint := "http://localhost:8080/registration/v1.0.0/login"

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return 1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1, fmt.Errorf("failed to register: %s", resp.Status)
	}
	var loginResponse LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		return 1, err
	}

	// parse url
	parsedUrl, err := url.Parse(loginResponse.URL)
	if err != nil {
		return 1, err
	}

	state := parsedUrl.Query().Get("state")
	if state == "" {
		return 1, fmt.Errorf("missing state in URL")
	}

	fmt.Printf("Login URL: %s\n", loginResponse.URL)

	fmt.Printf("Waiting for confirmation")
	found := false
	var token string
	for _ = range 30 {
		// poll for token
		pollUrl := fmt.Sprintf("http://localhost:8080/registration/v1.0.0/token?state=%s", state)
		resp, err := http.Get(pollUrl)
		if err != nil {
			return 1, err
		}

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusNotFound {
				fmt.Printf(".")
			} else {
				resp.Body.Close()
				return 1, fmt.Errorf("failed to poll token: %s", resp.Status)
			}
		} else {
			found = true
			var tokenResult struct {
				Token *oauth2.Token `json:"token"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&tokenResult); err != nil {
				resp.Body.Close()
				return 1, err
			}
			if tokenResult.Token != nil {
				token = tokenResult.Token.AccessToken
			}
			break
		}
		time.Sleep(time.Second)
	}
	if !found {
		fmt.Println()
		return 1, fmt.Errorf("token not found")
	}
	fmt.Println("\nAuthenticated with token", token)

	//fmt.Println(utils.GetVersion())
	return 0, nil
}
