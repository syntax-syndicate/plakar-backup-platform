package subcommands

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/vmihailenco/msgpack/v5"
)

type CommandFlags uint32

const (
	NeedRepositoryKey CommandFlags = 1 << iota
	BeforeRepositoryOpen
	AgentSupport
)

type Subcommand interface {
	Parse(ctx *appcontext.AppContext, args []string) error
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
	GetRepositorySecret() []byte
	GetFlags() CommandFlags
}

type SubcommandBase struct {
	RepositorySecret []byte
	Flags            CommandFlags
}

func (cmd *SubcommandBase) GetFlags() CommandFlags {
	return cmd.Flags
}

func (cmd *SubcommandBase) GetRepositorySecret() []byte {
	return cmd.RepositorySecret
}

type CmdFactory func() Subcommand
type subcmd struct {
	args    []string
	nargs   int
	factory CmdFactory
}

var subcommands []subcmd = make([]subcmd, 0)

func Register(factory CmdFactory, args ...string) {
	if len(args) == 0 {
		panic("can't register commands with zero arguments")
	}

	subcommands = append(subcommands, subcmd{
		args:    args,
		nargs:   len(args),
		factory: factory,
	})
}

func Lookup(arguments []string) (CmdFactory, []string, []string) {
	nargs := len(arguments)
	for _, subcmd := range subcommands {
		if nargs < subcmd.nargs {
			continue
		}

		if !slices.Equal(subcmd.args, arguments[:subcmd.nargs]) {
			continue
		}

		return subcmd.factory, arguments[:subcmd.nargs], arguments[subcmd.nargs:]
	}

	return nil, nil, nil
}

func List() []string {
	var list []string
	for _, command := range subcommands {
		list = append(list, strings.Join(command.args, " "))
	}
	sort.Strings(list)
	return list
}

// RPC extends subcommands.Subcommand, but it also includes the Name() method used to identify the RPC on decoding.
type RPC interface {
	Subcommand
	Name() string
}

type encodedRPC struct {
	Name        []string
	Subcommand  RPC
	StoreConfig map[string]string
}

// Encode marshals the RPC into the msgpack encoder. It prefixes the RPC with
// the Name() of the RPC. This is used to identify the RPC on decoding.
func EncodeRPC(encoder *msgpack.Encoder, name []string, cmd RPC, storeConfig map[string]string) error {
	return encoder.Encode(encodedRPC{
		Name:        name,
		Subcommand:  cmd,
		StoreConfig: storeConfig,
	})
}

// Decode extracts the request encoded by Encode(). It returns the name of the
// RPC and the raw bytes of the request. The raw bytes can be used by the caller
// to unmarshal the bytes with the correct struct.
func DecodeRPC(decoder *msgpack.Decoder) ([]string, map[string]string, []byte, error) {
	var request map[string]interface{}
	if err := decoder.Decode(&request); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode client request: %w", err)
	}

	rawRequest, err := msgpack.Marshal(request["Subcommand"])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal client request: %s", err)
	}

	// XXX: Not my proudest code lives below. We need to fix this later!
	tmp, ok := request["Name"].([]interface{})
	if !ok {
		return nil, nil, nil, fmt.Errorf("request does not contain a Name string field")
	}

	name := make([]string, 0, len(tmp))
	for _, elm := range tmp {
		tname, _ := elm.(string)
		name = append(name, tname)
	}

	storeConfig, ok := request["StoreConfig"].(map[string]interface{})
	if !ok {
		return nil, nil, nil, fmt.Errorf("request does not contain a StoreConfig field")
	}

	okStoreConfig := make(map[string]string)
	for k, v := range storeConfig {
		if str, ok := v.(string); ok {
			okStoreConfig[k] = str
		} else {
			return nil, nil, nil, fmt.Errorf("StoreConfig field %s is not a string", k)
		}
	}

	return name, okStoreConfig, rawRequest, nil
}
