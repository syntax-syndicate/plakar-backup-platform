package exec

import (
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
)

type Exec struct {
	RepositoryLocation string
	RepositorySecret   []byte

	SnapshotPrefix string
	Args           []string
}

func (cmd *Exec) Name() string {
	return "exec"
}

func (cmd *Exec) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snapshots, err := utils.GetSnapshots(repo, []string{cmd.SnapshotPrefix})
	if err != nil {
		log.Fatal(err)
	}
	if len(snapshots) != 1 {
		return 0, nil
	}
	snap := snapshots[0]
	defer snap.Close()

	_, pathname := utils.ParseSnapshotID(cmd.SnapshotPrefix)

	rd, err := snap.NewReader(pathname)
	if err != nil {
		ctx.GetLogger().Error("exec: %s: failed to open: %s", pathname, err)
		return 1, err
	}
	defer rd.Close()

	file, err := os.CreateTemp(os.TempDir(), "plakar")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())
	file.Chmod(0500)

	_, err = io.Copy(file, rd)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	p := exec.Command(file.Name(), cmd.Args...)
	stdin, err := p.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer stdin.Close()
		io.Copy(stdin, os.Stdin)
	}()

	stdout, err := p.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer stdout.Close()
		io.Copy(os.Stdout, stdout)
	}()

	stderr, err := p.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer stdout.Close()
		io.Copy(os.Stderr, stderr)
	}()
	if p.Start() == nil {
		p.Wait()
		return p.ProcessState.ExitCode(), nil
	}
	return 1, err
}
