package subcommands

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

type MockedSubcommand struct {
	name string
}

func (m MockedSubcommand) Name() string {
	return m.name
}

func (m MockedSubcommand) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return len(m.name), nil
}

func TestRegister(t *testing.T) {
	t.Cleanup(func() {
		// need to reset the global var between tests
		subcommands = make(map[string]parseArgsFn)
	})
	Register("test", func(*appcontext.AppContext, *repository.Repository, []string) (Subcommand, error) {
		return MockedSubcommand{}, nil
	})

	if _, exists := subcommands["test"]; !exists {
		t.Errorf("expected subcommand to be registered")
	}
}

func TestParse(t *testing.T) {
	t.Cleanup(func() {
		// need to reset the global var between tests
		subcommands = make(map[string]parseArgsFn)
	})

	Register("test", func(*appcontext.AppContext, *repository.Repository, []string) (Subcommand, error) {
		return MockedSubcommand{}, nil
	})

	ctx := &appcontext.AppContext{}
	repo := &repository.Repository{}
	cmd := "test"
	args := []string{}

	_, err := Parse(ctx, repo, "unknown", args)
	require.Error(t, err, "unknown command: unknown")

	subcmd, err := Parse(ctx, repo, cmd, args)
	require.NoError(t, err)
	require.NotNil(t, subcmd)
}

func TestList(t *testing.T) {
	t.Cleanup(func() {
		// need to reset the global var between tests
		subcommands = make(map[string]parseArgsFn)
	})

	Register("test1", func(*appcontext.AppContext, *repository.Repository, []string) (Subcommand, error) {
		return nil, nil
	})
	Register("test2", func(*appcontext.AppContext, *repository.Repository, []string) (Subcommand, error) {
		return nil, nil
	})

	list := List()
	if len(list) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(list))
	}

	if list[0] != "test1" || list[1] != "test2" {
		t.Errorf("expected subcommands to be sorted alphabetically")
	}
}

type MockedRPC struct {
	Subcommand
	name string
}

func (m MockedRPC) Name() string {
	return m.name
}

func TestEncodeDecodeRPC(t *testing.T) {
	subcmdIn := MockedSubcommand{name: "test"}
	rpc := MockedRPC{
		name:       "test",
		Subcommand: subcmdIn,
	}

	bufIn := bytes.NewBuffer(nil)
	encoder := msgpack.NewEncoder(bufIn)
	err := EncodeRPC(encoder, rpc)
	require.NoError(t, err)

	decoder := msgpack.NewDecoder(bufIn)
	name, rawRequest, err := DecodeRPC(decoder)
	require.NoError(t, err)
	require.Equal(t, "test", name)
	require.NotEmpty(t, rawRequest)

	var subcmdOut MockedSubcommand
	err = msgpack.Unmarshal(rawRequest, &subcmdOut)
	require.NoError(t, err)
	val, err := subcmdOut.Execute(nil, nil)
	require.NoError(t, err)
	require.Equal(t, 0, val)
}
