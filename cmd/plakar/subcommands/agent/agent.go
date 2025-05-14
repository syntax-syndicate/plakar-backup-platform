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
	"errors"
	"flag"
	"fmt"
	"io"
	"log/syslog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/maintenance"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/rm"
	syncSubcmd "github.com/PlakarKorp/plakar/cmd/plakar/subcommands/sync"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/reporting"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/scheduler"
	"github.com/PlakarKorp/plakar/services"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &AgentTasksConfigure{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen, "agent", "tasks", "configure")
	subcommands.Register(func() subcommands.Subcommand { return &AgentTasksStart{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen, "agent", "tasks", "start")
	subcommands.Register(func() subcommands.Subcommand { return &AgentTasksStop{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen, "agent", "tasks", "stop")
	subcommands.Register(func() subcommands.Subcommand { return &AgentRestart{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen|subcommands.IgnoreVersion, "agent", "restart")
	subcommands.Register(func() subcommands.Subcommand { return &AgentStop{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen|subcommands.IgnoreVersion, "agent", "stop")
	subcommands.Register(func() subcommands.Subcommand { return &Agent{} },
		subcommands.BeforeRepositoryOpen, "agent", "start")
	subcommands.Register(func() subcommands.Subcommand { return &Agent{} },
		subcommands.BeforeRepositoryOpen, "agent")
}

var agentContextSingleton *AgentContext

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

func (cmd *Agent) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_foreground bool
	var opt_logfile string

	flags := flag.NewFlagSet("agent", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&cmd.prometheus, "prometheus", "", "prometheus exporter interface, e.g. 127.0.0.1:9090")
	flags.BoolVar(&opt_foreground, "foreground", false, "run in foreground")
	flags.StringVar(&opt_logfile, "log", "", "log file")
	flags.Parse(args)

	if !opt_foreground && os.Getenv("REEXEC") == "" {
		err := daemonize(os.Args)
		return err
	}

	if opt_logfile != "" {
		f, err := os.OpenFile(opt_logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		ctx.GetLogger().SetOutput(f)
	} else if !opt_foreground {
		w, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "plakar")
		if err != nil {
			return err
		}
		ctx.GetLogger().SetSyslogOutput(w)
	}

	cmd.socketPath = filepath.Join(ctx.CacheDir, "agent.sock")

	ctx.GetLogger().Info("Plakar agent up")
	return nil
}

type schedulerState int8

var (
	AGENT_SCHEDULER_STOPPED schedulerState = 0
	AGENT_SCHEDULER_RUNNING schedulerState = 1
)

type AgentContext struct {
	agentCtx        *appcontext.AppContext
	schedulerCtx    *appcontext.AppContext
	schedulerConfig *scheduler.Configuration
	schedulerState  schedulerState
	mtx             sync.Mutex
}

type AgentStop struct {
	subcommands.SubcommandBase
}

func (cmd *AgentStop) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent stop", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	return nil
}

func (cmd *AgentStop) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	syscall.Kill(os.Getpid(), syscall.SIGINT)
	return 0, nil
}

type AgentRestart struct {
	subcommands.SubcommandBase
}

func (cmd *AgentRestart) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent restart", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	return nil
}

func (cmd *AgentRestart) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if err := restart(); err != nil {
		return 1, fmt.Errorf("failed to restart agent: %w", err)
	}
	return 0, nil
}

func restart() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable path: %w", err)
	}
	return syscall.Exec(exePath, append([]string{exePath}, os.Args[1:]...), os.Environ())
}

type Agent struct {
	subcommands.SubcommandBase

	prometheus string
	socketPath string

	listener net.Listener
}

func (cmd *Agent) checkSocket() bool {
	conn, err := net.Dial("unix", cmd.socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (cmd *Agent) Close() error {
	if cmd.listener != nil {
		cmd.listener.Close()
	}
	if err := os.Remove(cmd.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func isDisconnectError(err error) bool {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (cmd *Agent) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	agentContextSingleton = &AgentContext{
		agentCtx: ctx,
	}

	if err := cmd.ListenAndServe(ctx); err != nil {
		return 1, err
	}
	ctx.GetLogger().Info("Server gracefully stopped")
	return 0, nil
}

func (cmd *Agent) ListenAndServe(ctx *appcontext.AppContext) error {
	if _, err := os.Stat(cmd.socketPath); err == nil {
		if !cmd.checkSocket() {
			cmd.Close()
		} else {
			return fmt.Errorf("already running")
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	var err error

	// Bind socket
	cmd.listener, err = net.Listen("unix", cmd.socketPath)
	if err != nil {
		return fmt.Errorf("failed to bind socket: %w", err)
	}
	defer os.Remove(cmd.socketPath)

	// Set socket permissions
	if err := os.Chmod(cmd.socketPath, 0600); err != nil {
		cmd.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	var promlistener net.Listener
	if cmd.prometheus != "" {
		promlistener, err = net.Listen("tcp", cmd.prometheus)
		if err != nil {
			return fmt.Errorf("failed to bind prometheus listener: %w", err)
		}
		defer promlistener.Close()

		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.Serve(promlistener, nil)
		}()
	}

	// close the listener when the context gets closed
	go func() {
		<-ctx.Done()
		if promlistener != nil {
			promlistener.Close()
		}
		cmd.listener.Close()
	}()

	var wg sync.WaitGroup

	for {
		conn, err := cmd.listener.Accept()
		if err != nil {
			wg.Wait()
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return nil
			}
			ctx.GetLogger().Warn("failed to accept connection: %v", err)
			return err
		}

		wg.Add(1)
		go handleClient(ctx, &wg, conn)
	}
}

func handleClient(ctx *appcontext.AppContext, wg *sync.WaitGroup, conn net.Conn) {
	defer conn.Close()
	defer wg.Done()

	mu := sync.Mutex{}

	var encodingErrorOccurred bool
	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	clientContext := appcontext.NewAppContextFrom(ctx)
	defer clientContext.Close()

	// handshake
	var (
		clientvers []byte
		ourvers    = []byte(utils.GetVersion())
	)
	if err := decoder.Decode(&clientvers); err != nil {
		return
	}
	if err := encoder.Encode(ourvers); err != nil {
		return
	}

	write := func(packet agent.Packet) {
		if encodingErrorOccurred {
			return
		}
		select {
		case <-clientContext.Done():
			return
		default:
			mu.Lock()
			if err := encoder.Encode(&packet); err != nil {
				encodingErrorOccurred = true
				ctx.GetLogger().Warn("client write error: %v", err)
			}
			mu.Unlock()
		}
	}
	read := func(v interface{}) (interface{}, error) {
		if err := decoder.Decode(v); err != nil {
			if isDisconnectError(err) {
				clientContext.Close()
			}
			return nil, err
		}
		return v, nil
	}

	processStdout := func(data string) {
		write(agent.Packet{
			Type: "stdout",
			Data: []byte(data),
		})
	}

	processStderr := func(data string) {
		write(agent.Packet{
			Type: "stderr",
			Data: []byte(data),
		})
	}

	clientContext.Stdout = &CustomWriter{processFunc: processStdout}
	clientContext.Stderr = &CustomWriter{processFunc: processStderr}

	logger := logging.NewLogger(clientContext.Stdout, clientContext.Stderr)
	logger.EnableInfo()
	clientContext.SetLogger(logger)

	name, storeConfig, request, err := subcommands.DecodeRPC(decoder)
	if err != nil {
		if isDisconnectError(err) {
			ctx.GetLogger().Warn("client disconnected during initial request")
			return
		}
		ctx.GetLogger().Warn("Failed to decode RPC: %v", err)
		fmt.Fprintf(clientContext.Stderr, "%s\n", err)
		return
	}

	// Attempt another decode to detect client disconnection during processing
	go func() {
		var tmp interface{}
		read(&tmp)
	}()

	subcommand, _, _ := subcommands.Lookup(name)
	if subcommand == nil {
		ctx.GetLogger().Warn("unknown command received: %s", name)
		fmt.Fprintf(clientContext.Stderr, "unknown command received %s\n", name)
		return
	}
	if err := msgpack.Unmarshal(request, &subcommand); err != nil {
		ctx.GetLogger().Warn("Failed to decode client request: %v", err)
		fmt.Fprintf(clientContext.Stderr, "Failed to decode client request: %s\n", err)
		return
	}

	if subcommand.GetLogInfo() {
		clientContext.GetLogger().EnableInfo()
	}
	clientContext.GetLogger().EnableTracing(subcommand.GetLogTraces())

	ctx.GetLogger().Info("%s on %s", name, storeConfig["location"])

	var store storage.Store
	var repo *repository.Repository

	if subcommand.GetFlags()&subcommands.BeforeRepositoryOpen != 0 {
		// nop
	} else if subcommand.GetFlags()&subcommands.BeforeRepositoryWithStorage != 0 {
		repo, err = repository.Inexistent(clientContext, storeConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open raw storage: %v", err)
			fmt.Fprintf(clientContext.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
			return
		}
		defer repo.Close()
	} else {
		var serializedConfig []byte
		clientContext.SetSecret(subcommand.GetRepositorySecret())
		store, serializedConfig, err = storage.Open(clientContext, storeConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open storage: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to open storage: %s\n", err)
			return
		}
		defer store.Close()

		repo, err = repository.New(clientContext, store, serializedConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open repository: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to open repository: %s\n", err)
			return
		}
		defer repo.Close()
	}

	eventsDone := make(chan struct{})
	eventsChan := clientContext.Events().Listen()
	go func() {
		for evt := range eventsChan {
			serialized, err := events.Serialize(evt)
			if err != nil {
				clientContext.GetLogger().Warn("Failed to serialize event: %v", err)
				fmt.Fprintf(clientContext.Stderr, "Failed to serialize event: %s\n", err)
				return
			}
			// Send the event to the client
			write(agent.Packet{
				Type: "event",
				Data: serialized,
			})
		}
		eventsDone <- struct{}{}
	}()

	var taskKind string
	switch subcommand.(type) {
	case *backup.Backup:
		taskKind = "backup"
	case *check.Check:
		taskKind = "check"
	case *restore.Restore:
		taskKind = "restore"
	case *syncSubcmd.Sync:
		taskKind = "sync"
	case *rm.Rm:
		taskKind = "rm"
	case *maintenance.Maintenance:
		taskKind = "maintenance"
	}

	var doReport bool
	if repo != nil && taskKind != "" {
		authToken, err := clientContext.GetAuthToken(repo.Configuration().RepositoryID)
		if err == nil && authToken != "" {
			sc := services.NewServiceConnector(clientContext, authToken)
			enabled, err := sc.GetServiceStatus("alerting")
			if err == nil && enabled {
				doReport = true
			}
		}
	}

	reporter := reporting.NewReporter(doReport, repo, ctx.GetLogger())
	reporter.TaskStart(taskKind, "@agent")
	reporter.WithRepositoryName(storeConfig["location"])
	if repo != nil {
		reporter.WithRepository(repo)
	}

	var status int
	var snapshotID objects.MAC
	var warning error
	if _, ok := subcommand.(*backup.Backup); ok {
		subcommand := subcommand.(*backup.Backup)
		status, err, snapshotID, warning = subcommand.DoBackup(clientContext, repo)
		if err == nil {
			reporter.WithSnapshotID(snapshotID)
		}
	} else {
		status, err = subcommand.Execute(clientContext, repo)
	}

	if status == 0 {
		if warning != nil {
			reporter.TaskWarning("warning: %s", warning)
		} else {
			reporter.TaskDone()
		}
	} else if err != nil {
		reporter.TaskFailed(0, "error: %s", err)
	}

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	write(agent.Packet{
		Type:     "exit",
		ExitCode: status,
		Err:      errStr,
	})

	clientContext.Close()
	<-eventsDone
}

type CustomWriter struct {
	processFunc func(string) // Function to handle the log lines
}

// Write implements the `io.Writer` interface.
func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	cw.processFunc(string(p))
	return len(p), nil
}
