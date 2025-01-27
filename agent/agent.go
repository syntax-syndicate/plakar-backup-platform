package agent

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/vmihailenco/msgpack/v5"
)

type Agent struct {
	socketPath string
	listener   net.Listener
	ctx        *appcontext.AppContext
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
}

func NewAgent(ctx *appcontext.AppContext, network string, address string) (*Agent, error) {
	if network != "unix" {
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	d := &Agent{
		socketPath: address,
		ctx:        ctx,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	if _, err := os.Stat(d.socketPath); err == nil {
		if !d.checkSocket() {
			d.Close()
		} else {
			return nil, fmt.Errorf("already running")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return nil, err
	}
	d.listener = listener

	if err := os.Chmod(d.socketPath, 0600); err != nil {
		d.Close()
		return nil, err
	}

	return d, nil
}

func (d *Agent) checkSocket() bool {
	conn, err := net.Dial("unix", d.socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (d *Agent) Close() error {
	if d.listener != nil {
		d.listener.Close()
	}
	if err := os.Remove(d.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (d *Agent) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.listener != nil {
		if err := d.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
		d.listener = nil
	}

	// Wait for all active connections or until the context is done
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All connections gracefully closed
	case <-ctx.Done():
		// Context canceled or timed out
		return ctx.Err()
	}

	return d.Close()
}

type CustomWriter struct {
	processFunc func(string) // Function to handle the log lines
}

// Write implements the `io.Writer` interface.
func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	cw.processFunc(string(p))
	return len(p), nil
}

func (d *Agent) ListenAndServe(handler func(*appcontext.AppContext, net.Conn) (int, error)) error {
	for {
		select {
		case <-d.cancelCtx.Done():
			return nil
		default:
		}

		conn, err := d.listener.Accept()
		if err != nil {
			select {
			case <-d.cancelCtx.Done():
				return nil
			default:
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					return nil
				}
				return fmt.Errorf("failed to accept connection: %w", err)
			}
		}

		d.wg.Add(1)
		go func(_conn net.Conn) {
			defer _conn.Close()
			defer d.wg.Done()

			encoder := msgpack.NewEncoder(_conn)
			var encodingErrorOccurred bool

			processStdout := func(line string) {
				if encodingErrorOccurred {
					return
				}
				response := Packet{
					Type:   "stdout",
					Output: line,
				}
				if err := encoder.Encode(&response); err != nil {
					fmt.Println("failed to encode response:", err)
					encodingErrorOccurred = true
				}
			}

			processStderr := func(line string) {
				if encodingErrorOccurred {
					return
				}
				response := Packet{
					Type:   "stderr",
					Output: line,
				}
				if err := encoder.Encode(&response); err != nil {
					fmt.Println("failed to encode response:", err)
					encodingErrorOccurred = true
				}
			}

			d.ctx.SetStdout(&CustomWriter{processFunc: processStdout})
			d.ctx.SetStderr(&CustomWriter{processFunc: processStderr})

			logger := logging.NewLogger(d.ctx.Stdout(), d.ctx.Stdout())
			logger.EnableInfo()
			d.ctx.SetLogger(logger)

			status, err := handler(d.ctx, _conn)
			response := Packet{
				Type:     "exit",
				ExitCode: status,
				Err:      fmt.Sprintf("%v", err),
			}
			if err := encoder.Encode(&response); err != nil {
				fmt.Println("failed to encode response:", err)
			}
		}(conn)
	}
}

// Client structure and other code remain unchanged

type CommandRequest struct {
	AppContext *appcontext.AppContext
	Repository string
	Cmd        string
	Argv       []string
}

type Packet struct {
	Type     string
	Output   string
	ExitCode int
	Err      string
}

type Client struct {
	conn net.Conn
}

func NewClient(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}

	return &Client{conn: conn}, nil
}

func (c *Client) SendCommand(ctx *appcontext.AppContext, repo string, cmd string, argv []string) (int, error) {
	encoder := msgpack.NewEncoder(c.conn)
	decoder := msgpack.NewDecoder(c.conn)

	request := CommandRequest{
		AppContext: ctx,
		Repository: repo,
		Cmd:        cmd,
		Argv:       argv,
	}

	if err := encoder.Encode(&request); err != nil {
		return 1, fmt.Errorf("failed to encode command: %w", err)
	}

	var response Packet
	for {
		if err := decoder.Decode(&response); err != nil {
			return 1, fmt.Errorf("failed to decode response: %w", err)
		}
		switch response.Type {
		case "stdout":
			fmt.Printf("%s", response.Output)
		case "stderr":
			fmt.Fprintf(os.Stderr, "%s", response.Output)
		case "exit":
			return response.ExitCode, fmt.Errorf("%s", response.Err)
		}
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}
