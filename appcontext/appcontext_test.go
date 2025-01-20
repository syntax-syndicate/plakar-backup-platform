package appcontext

import (
	"testing"

	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/encryption/keypair"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestContext_SettersAndGetters(t *testing.T) {
	ctx := NewAppContext()

	defaultLogger := logging.NewLogger(nil, nil)
	defaultCachingManager := caching.NewManager("/tmp/test_plakar")
	defaultKeyPair, err := keypair.Generate()
	require.NoError(t, err)

	tests := []struct {
		name     string
		setter   func()
		getter   func() interface{}
		expected interface{}
	}{
		{
			name: "SetNumCPU",
			setter: func() {
				ctx.SetNumCPU(4)
			},
			getter:   func() interface{} { return ctx.GetNumCPU() },
			expected: 4,
		},
		{
			name: "SetUsername",
			setter: func() {
				ctx.SetUsername("testuser")
			},
			getter:   func() interface{} { return ctx.GetUsername() },
			expected: "testuser",
		},
		{
			name: "SetHostname",
			setter: func() {
				ctx.SetHostname("testhost")
			},
			getter:   func() interface{} { return ctx.GetHostname() },
			expected: "testhost",
		},
		{
			name: "SetCommandLine",
			setter: func() {
				ctx.SetCommandLine("test command line")
			},
			getter:   func() interface{} { return ctx.GetCommandLine() },
			expected: "test command line",
		},
		{
			name: "SetMachineID",
			setter: func() {
				ctx.SetMachineID("machine-123")
			},
			getter:   func() interface{} { return ctx.GetMachineID() },
			expected: "machine-123",
		},
		{
			name: "SetKeyFromFile",
			setter: func() {
				ctx.SetKeyFromFile("key123")
			},
			getter:   func() interface{} { return ctx.GetKeyFromFile() },
			expected: "key123",
		},
		{
			name: "SetHomeDir",
			setter: func() {
				ctx.SetHomeDir("/home/testuser")
			},
			getter:   func() interface{} { return ctx.GetHomeDir() },
			expected: "/home/testuser",
		},
		{
			name: "SetCacheDir",
			setter: func() {
				ctx.SetCacheDir("/cache/testuser")
			},
			getter:   func() interface{} { return ctx.GetCacheDir() },
			expected: "/cache/testuser",
		},
		{
			name: "SetOperatingSystem",
			setter: func() {
				ctx.SetOperatingSystem("linux")
			},
			getter:   func() interface{} { return ctx.GetOperatingSystem() },
			expected: "linux",
		},
		{
			name: "SetArchitecture",
			setter: func() {
				ctx.SetArchitecture("amd64")
			},
			getter:   func() interface{} { return ctx.GetArchitecture() },
			expected: "amd64",
		},
		{
			name: "SetProcessID",
			setter: func() {
				ctx.SetProcessID(12345)
			},
			getter:   func() interface{} { return ctx.GetProcessID() },
			expected: 12345,
		},
		{
			name: "SetKeyringDir",
			setter: func() {
				ctx.SetKeyringDir("/keyring/dir")
			},
			getter:   func() interface{} { return ctx.GetKeyringDir() },
			expected: "/keyring/dir",
		},
		{
			name: "SetIdentity",
			setter: func() {
				ctx.SetIdentity(uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"))
			},
			getter:   func() interface{} { return ctx.GetIdentity() },
			expected: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		},
		{
			name: "SetKeypair",
			setter: func() {
				ctx.SetKeypair(defaultKeyPair)
			},
			getter:   func() interface{} { return ctx.GetKeypair() },
			expected: defaultKeyPair,
		},
		{
			name: "SetPlakarClient",
			setter: func() {
				ctx.SetPlakarClient("plakar-client")
			},
			getter:   func() interface{} { return ctx.GetPlakarClient() },
			expected: "plakar-client",
		},
		{
			name: "SetMaxConcurrency",
			setter: func() {
				ctx.SetMaxConcurrency(10)
			},
			getter:   func() interface{} { return ctx.GetMaxConcurrency() },
			expected: 10,
		},
		{
			name: "SetCache",
			setter: func() {
				ctx.SetCache(defaultCachingManager)
			},
			getter:   func() interface{} { return ctx.GetCache() },
			expected: defaultCachingManager,
		},
		{
			name: "SetLogger",
			setter: func() {
				ctx.SetLogger(defaultLogger)
			},
			getter:   func() interface{} { return ctx.GetLogger() },
			expected: defaultLogger,
		},
		{
			name: "SetCWD",
			setter: func() {
				ctx.SetCWD("/current/working/dir")
			},
			getter:   func() interface{} { return ctx.GetCWD() },
			expected: "/current/working/dir",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.setter()
			if result := test.getter(); result != test.expected {
				t.Errorf("%s failed: expected %#v, got %#v", test.name, test.expected, result)
			}
		})
	}
}

func TestAppContextCloseEvents(t *testing.T) {
	ctx := NewAppContext()
	events := ctx.Events()
	if events == nil {
		t.Errorf("events is nil")
	}
	ctx.Close()
	// Check if events is closed
	select {
	case <-events.Listen():
		t.Errorf("events is not closed")
	default:
		// events is closed
	}
}
