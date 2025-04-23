package agent

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"

	"github.com/PlakarKorp/plakar/kloset/appcontext"
	"github.com/PlakarKorp/plakar/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/vmihailenco/msgpack/v5"
)

type Packet struct {
	Type     string
	Data     []byte
	ExitCode int
	Err      string
}

type Client struct {
	conn net.Conn
	enc  *msgpack.Encoder
	dec  *msgpack.Decoder
}

var (
	ErrRetryAgentless = errors.New("Failed to connect to agent, retry agentless")
	ErrWrongVersion   = errors.New("agent has a different version")
)

func ExecuteRPC(ctx *appcontext.AppContext, name []string, cmd subcommands.Subcommand, storeConfig map[string]string) (int, error) {
	rpcCmd, ok := cmd.(subcommands.RPC)
	if !ok {
		return 1, fmt.Errorf("subcommand is not an RPC")
	}

	client, err := NewClient(filepath.Join(ctx.CacheDir, "agent.sock"))
	if err != nil {
		if errors.Is(err, ErrWrongVersion) {
			ctx.GetLogger().Warn("%v", err)
		}
		ctx.GetLogger().Warn("failed to connect to agent, falling back to -no-agent")
		return 1, ErrRetryAgentless
	}
	defer client.Close()

	if status, err := client.SendCommand(ctx, name, rpcCmd, storeConfig); err != nil {
		return status, err
	}
	return 0, nil
}

func NewClient(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	c := &Client{
		conn: conn,
		enc:  encoder,
		dec:  decoder,
	}

	if err := c.handshake(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) handshake() error {
	ourvers := []byte(utils.GetVersion())

	if err := c.enc.Encode(ourvers); err != nil {
		return err
	}

	var agentvers []byte
	if err := c.dec.Decode(&agentvers); err != nil {
		return err
	}

	if !slices.Equal(ourvers, agentvers) {
		return fmt.Errorf("%w (%v)", ErrWrongVersion, string(agentvers))
	}

	return nil
}

func (c *Client) SendCommand(ctx *appcontext.AppContext, name []string, cmd subcommands.RPC, storeConfig map[string]string) (int, error) {
	if err := subcommands.EncodeRPC(c.enc, name, cmd, storeConfig); err != nil {
		return 1, err
	}

	var response Packet
	for {
		if err := c.dec.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return 1, fmt.Errorf("failed to decode response: %w", err)
		}
		switch response.Type {
		case "stdout":
			fmt.Printf("%s", string(response.Data))
		case "stderr":
			fmt.Fprintf(os.Stderr, "%s", string(response.Data))
		case "event":
			evt, err := events.Deserialize(response.Data)
			if err != nil {
				return 1, fmt.Errorf("failed to deserialize event: %w", err)
			}
			ctx.Events().Send(evt)
		case "exit":
			var err error
			if response.Err != "" {
				err = fmt.Errorf("%s", response.Err)
			}
			return response.ExitCode, err
		}
	}
	return 0, nil
}
func (c *Client) Close() error {
	return c.conn.Close()
}
