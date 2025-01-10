package appcontext

import (
	"testing"
)

func TestContext_SettersAndGetters(t *testing.T) {
	ctx := NewAppContext()

	tests := []struct {
		name     string
		setter   func()
		getter   func() interface{}
		expected interface{}
	}{
		{
			name: "SetNumCPU",
			setter: func() {
				ctx.NumCPU = 4
			},
			getter:   func() interface{} { return ctx.NumCPU },
			expected: 4,
		},
		{
			name: "SetUsername",
			setter: func() {
				ctx.Username = "testuser"
			},
			getter:   func() interface{} { return ctx.Username },
			expected: "testuser",
		},
		{
			name: "SetHostname",
			setter: func() {
				ctx.Hostname = "testhost"
			},
			getter:   func() interface{} { return ctx.Hostname },
			expected: "testhost",
		},
		{
			name: "SetCommandLine",
			setter: func() {
				ctx.CommandLine = "test command line"
			},
			getter:   func() interface{} { return ctx.CommandLine },
			expected: "test command line",
		},
		{
			name: "SetMachineID",
			setter: func() {
				ctx.MachineID = "machine-123"
			},
			getter:   func() interface{} { return ctx.MachineID },
			expected: "machine-123",
		},
		{
			name: "SetKeyFromFile",
			setter: func() {
				ctx.KeyFromFile = "key123"
			},
			getter:   func() interface{} { return ctx.KeyFromFile },
			expected: "key123",
		},
		{
			name: "SetHomeDir",
			setter: func() {
				ctx.HomeDir = "/home/testuser"
			},
			getter:   func() interface{} { return ctx.HomeDir },
			expected: "/home/testuser",
		},
		{
			name: "SetCacheDir",
			setter: func() {
				ctx.CacheDir = "/cache/testuser"
			},
			getter:   func() interface{} { return ctx.CacheDir },
			expected: "/cache/testuser",
		},
		{
			name: "SetOperatingSystem",
			setter: func() {
				ctx.OperatingSystem = "linux"
			},
			getter:   func() interface{} { return ctx.OperatingSystem },
			expected: "linux",
		},
		{
			name: "SetArchitecture",
			setter: func() {
				ctx.Architecture = "amd64"
			},
			getter:   func() interface{} { return ctx.Architecture },
			expected: "amd64",
		},
		{
			name: "SetProcessID",
			setter: func() {
				ctx.ProcessID = 12345
			},
			getter:   func() interface{} { return ctx.ProcessID },
			expected: 12345,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.setter()
			if result := test.getter(); result != test.expected {
				t.Errorf("%s failed: expected %v, got %v", test.name, test.expected, result)
			}
		})
	}
}
