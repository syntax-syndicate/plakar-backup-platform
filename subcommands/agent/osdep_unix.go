//go:build !windows

package agent

import (
	"fmt"
	"os"
	"syscall"
	"log/syslog"

	"github.com/PlakarKorp/plakar/appcontext"
)

func setupSyslog(ctx *appcontext.AppContext) error {
	w, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "plakar")
	if err != nil {
		return err
	}
	ctx.GetLogger().SetSyslogOutput(w)
	return nil
}

func daemonize(argv []string) error {
	binary, err := os.Executable()
	if err != nil {
		return err
	}

	procAttr := syscall.ProcAttr{}
	procAttr.Files = []uintptr{
		uintptr(syscall.Stdin),
		uintptr(syscall.Stdout),
		uintptr(syscall.Stderr),
	}
	procAttr.Env = append(os.Environ(),
		"REEXEC=1",
	)

	pid, err := syscall.ForkExec(binary, argv, &procAttr)
	if err != nil {
		return err
	}
	fmt.Printf("agent started with pid=%d\n", pid)
	os.Exit(0)
	return nil
}

func stop() error {
	return syscall.Kill(os.Getpid(), syscall.SIGINT)
}
