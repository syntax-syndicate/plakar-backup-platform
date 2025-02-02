package ui

import (
	"fmt"
	"os"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
	v2 "github.com/PlakarKorp/plakar/ui/v2"
	"github.com/google/uuid"
)

type Ui struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Addr    string
	Cors    bool
	NoAuth  bool
	NoSpawn bool
}

func (cmd *Ui) Name() string {
	return "ui"
}

func (cmd *Ui) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	ui_opts := v2.UiOptions{
		NoSpawn: cmd.NoSpawn,
		Cors:    cmd.Cors,
		Token:   "",
	}

	if !cmd.NoAuth {
		ui_opts.Token = uuid.NewString()
	}

	err := v2.Ui(repo, cmd.Addr, &ui_opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ui: %s\n", err)
		return 1, err
	}
	return 0, err
}
