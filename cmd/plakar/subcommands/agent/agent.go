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
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc"
	"github.com/PlakarKorp/plakar/rpc/archive"
	"github.com/PlakarKorp/plakar/rpc/backup"
	"github.com/PlakarKorp/plakar/rpc/cat"
	"github.com/PlakarKorp/plakar/rpc/check"
	"github.com/PlakarKorp/plakar/rpc/checksum"
	"github.com/PlakarKorp/plakar/rpc/cleanup"
	"github.com/PlakarKorp/plakar/rpc/clone"
	"github.com/PlakarKorp/plakar/rpc/diff"
	"github.com/PlakarKorp/plakar/rpc/exec"
	"github.com/PlakarKorp/plakar/rpc/info"
	"github.com/PlakarKorp/plakar/rpc/locate"
	"github.com/PlakarKorp/plakar/rpc/ls"
	"github.com/PlakarKorp/plakar/rpc/mount"
	"github.com/PlakarKorp/plakar/rpc/restore"
	"github.com/PlakarKorp/plakar/rpc/rm"
	"github.com/PlakarKorp/plakar/rpc/server"
	cmd_sync "github.com/PlakarKorp/plakar/rpc/sync"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vmihailenco/msgpack/v5"
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

	return &Agent{
		Prometheus: opt_prometheus,
		SocketPath: opt_socketPath,
	}, nil
}

type Agent struct {
	Prometheus string
	SocketPath string

	listener net.Listener
}

func (cmd *Agent) Name() string {
	return "agent"
}

type Packet struct {
	Type     string
	Data     []byte
	ExitCode int
	Err      string
}

func (cmd *Agent) checkSocket() bool {
	conn, err := net.Dial("unix", cmd.SocketPath)
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
	if err := os.Remove(cmd.SocketPath); err != nil && !os.IsNotExist(err) {
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
	if err := cmd.ListenAndServe(ctx); err != nil {
		return 1, err
	}
	log.Println("Server gracefully stopped")
	return 0, nil
}

func (cmd *Agent) ListenAndServe(ctx *appcontext.AppContext) error {
	if _, err := os.Stat(cmd.SocketPath); err == nil {
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
	cmd.listener, err = net.Listen("unix", cmd.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to bind socket: %w", err)
	}
	defer os.Remove(cmd.SocketPath)

	// Set socket permissions
	if err := os.Chmod(cmd.SocketPath, 0600); err != nil {
		cmd.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	cancelCtx, _ := context.WithCancel(context.Background())

	// XXX: start prom logic
	if cmd.Prometheus != "" {
	}

	var wg sync.WaitGroup

	for {
		select {
		case <-cancelCtx.Done():
			return nil
		default:
		}

		conn, err := cmd.listener.Accept()
		if err != nil {
			select {
			case <-cancelCtx.Done():
				return nil
			default:
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					return nil
				}
				return fmt.Errorf("failed to accept connection: %w", err)
			}
		}

		wg.Add(1)

		go func(_conn net.Conn) {
			defer _conn.Close()
			defer wg.Done()

			mu := sync.Mutex{}

			encoder := msgpack.NewEncoder(_conn)
			var encodingErrorOccurred bool

			// Create a context tied to the connection
			cancelCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			clientContext := appcontext.NewAppContextFrom(ctx)
			clientContext.SetContext(cancelCtx)
			defer clientContext.Close()

			processStdout := func(data string) {
				if encodingErrorOccurred {
					return
				}
				select {
				case <-clientContext.GetContext().Done():
					return
				default:
					response := Packet{
						Type: "stdout",
						Data: []byte(data),
					}
					mu.Lock()
					if err := encoder.Encode(&response); err != nil {
						encodingErrorOccurred = true
					}
					mu.Unlock()
				}
			}

			processStderr := func(data string) {
				if encodingErrorOccurred {
					return
				}

				select {
				case <-clientContext.GetContext().Done():
					return
				default:
					response := Packet{
						Type: "stderr",
						Data: []byte(data),
					}
					mu.Lock()
					if err := encoder.Encode(&response); err != nil {
						encodingErrorOccurred = true
					}
					mu.Unlock()

				}
			}

			clientContext.Stdout = &CustomWriter{processFunc: processStdout}
			clientContext.Stderr = &CustomWriter{processFunc: processStderr}

			logger := logging.NewLogger(clientContext.Stdout, clientContext.Stderr)
			logger.EnableInfo()
			clientContext.SetLogger(logger)

			decoder := msgpack.NewDecoder(conn)

			if err != nil {
				if isDisconnectError(err) {
					fmt.Fprintf(os.Stderr, "Client disconnected during initial request\n")
					cancel() // Cancel the context on disconnect
					return
				}
			}

			// Decode the client request
			var request map[string]interface{}
			if err := decoder.Decode(&request); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
				return
			}

			// Remarshal the request to get the raw bytes. This is necessary
			// because we can't rewind the decoder, and we need to redecode the
			// data below to the correct struct.
			rawRequest, err := msgpack.Marshal(request)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to marshal client request: %s\n", err)
				return
			}

			var subcommand rpc.RPC
			var repositoryLocation string
			var repositorySecret []byte

			switch request["Name"] {
			case "cat":
				var cmd struct {
					Name       string
					Subcommand cat.Cat
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "ls":
				var cmd struct {
					Name       string
					Subcommand ls.Ls
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "backup":
				var cmd struct {
					Name       string
					Subcommand backup.Backup
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "info_repository":
				var cmd struct {
					Name       string
					Subcommand info.InfoRepository
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "info_snapshot":
				var cmd struct {
					Name       string
					Subcommand info.InfoSnapshot
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "info_errors":
				var cmd struct {
					Name       string
					Subcommand info.InfoErrors
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "info_state":
				var cmd struct {
					Name       string
					Subcommand info.InfoState
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "info_packfile":
				var cmd struct {
					Name       string
					Subcommand info.InfoPackfile
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "info_object":
				var cmd struct {
					Name       string
					Subcommand info.InfoObject
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "info_vfs":
				var cmd struct {
					Name       string
					Subcommand info.InfoVFS
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "rm":
				var cmd struct {
					Name       string
					Subcommand rm.Rm
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "checksum":
				var cmd struct {
					Name       string
					Subcommand checksum.Checksum
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "locate":
				var cmd struct {
					Name       string
					Subcommand locate.Locate
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "check":
				var cmd struct {
					Name       string
					Subcommand check.Check
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "cleanup":
				var cmd struct {
					Name       string
					Subcommand cleanup.Cleanup
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "clone":
				var cmd struct {
					Name       string
					Subcommand clone.Clone
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "archive":
				var cmd struct {
					Name       string
					Subcommand archive.Archive
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "diff":
				var cmd struct {
					Name       string
					Subcommand diff.Diff
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "exec":
				var cmd struct {
					Name       string
					Subcommand exec.Exec
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "mount":
				var cmd struct {
					Name       string
					Subcommand mount.Mount
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "restore":
				var cmd struct {
					Name       string
					Subcommand restore.Restore
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "server":
				var cmd struct {
					Name       string
					Subcommand server.Server
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			case "sync":
				var cmd struct {
					Name       string
					Subcommand cmd_sync.Sync
				}
				if err := msgpack.Unmarshal(rawRequest, &cmd); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
					return
				}
				subcommand = &cmd.Subcommand
				repositoryLocation = cmd.Subcommand.RepositoryLocation
				repositorySecret = cmd.Subcommand.RepositorySecret
			default:
				fmt.Fprintf(os.Stderr, "Unknown RPC: %s\n", request["Name"])
				return
			}

			var repo *repository.Repository

			if repositoryLocation != "" {
				store, err := storage.Open(repositoryLocation)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open storage: %s\n", err)
					return
				}
				defer store.Close()

				repo, err = repository.New(clientContext, store, clientContext.GetSecret())
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open repository: %s\n", err)
					return
				}
				defer repo.Close()
			}

			if repositorySecret != nil {
				clientContext.SetSecret(repositorySecret)
			}

			eventsDone := make(chan struct{})
			eventsChan := clientContext.Events().Listen()
			go func() {
				for evt := range eventsChan {
					serialized, err := events.Serialize(evt)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to serialize event: %s\n", err)
						return
					}
					// Send the event to the client
					response := Packet{
						Type: "event",
						Data: serialized,
					}
					select {
					case <-clientContext.GetContext().Done():
						return
					default:
						mu.Lock()
						err = encoder.Encode(&response)
						mu.Unlock()
						if err != nil {
							fmt.Fprintf(os.Stderr, "Failed to encode event: %s\n", err)
							return
						}
					}
				}
				eventsDone <- struct{}{}
			}()

			status, err := subcommand.Execute(clientContext, repo)

			clientContext.Close()
			<-eventsDone

			select {
			case <-clientContext.GetContext().Done():
				return
			default:
				errStr := ""
				if err != nil {
					errStr = err.Error()
				}
				response := Packet{
					Type:     "exit",
					ExitCode: status,
					Err:      errStr,
				}
				mu.Lock()
				if err := encoder.Encode(&response); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to encode response: %s\n", err)
				}
				mu.Unlock()
			}

		}(conn)
	}
}

type CustomWriter struct {
	processFunc func(string) // Function to handle the log lines
}

// Write implements the `io.Writer` interface.
func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	cw.processFunc(string(p))
	return len(p), nil
}

// Prometheus part
var (
	// Define a counter
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "my_exporter_requests_total",
			Help: "Total number of processed requests",
		},
		[]string{"method", "status"},
	)

	// Define a gauge
	upGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "my_exporter_up",
			Help: "Exporter up status",
		},
	)

	disconnectsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "my_exporter_disconnects_total",
			Help: "Total number of client disconnections",
		},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(upGauge)
	prometheus.MustRegister(disconnectsTotal)
}

func trackRequest(method, status string) {
	requestsTotal.WithLabelValues(method, status).Inc()
}

func setUpGauge(value float64) {
	upGauge.Set(value)
}

func handleDisconnect() {
	disconnectsTotal.Inc() // Increment disconnect counter
}
