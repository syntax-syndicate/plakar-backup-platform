package appcontext

import (
	"context"
	"io"
	"os"

	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/encryption/keypair"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/google/uuid"
)

type AppContext struct {
	events  *events.Receiver
	cache   *caching.Manager
	logger  *logging.Logger
	context context.Context
	secret  []byte

	stdout io.Writer
	stderr io.Writer

	NumCPU      int
	Username    string
	HomeDir     string
	Hostname    string
	CommandLine string
	MachineID   string
	KeyFromFile string
	CacheDir    string
	KeyringDir  string

	OperatingSystem string
	Architecture    string
	ProcessID       int

	Client string

	CWD            string
	MaxConcurrency int

	Identity uuid.UUID
	Keypair  *keypair.KeyPair
}

func NewAppContext() *AppContext {
	return &AppContext{
		events:  events.New(),
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		context: context.Background(),
	}
}

func NewAppContextFrom(template *AppContext) *AppContext {
	ctx := NewAppContext()
	events := ctx.events
	*ctx = *template
	ctx.SetCache(template.GetCache())
	ctx.SetLogger(template.GetLogger())
	ctx.events = events
	return ctx
}

func (c *AppContext) Close() {
	c.events.Close()
}

func (c *AppContext) Events() *events.Receiver {
	return c.events
}

func (c *AppContext) SetCache(cacheManager *caching.Manager) {
	c.cache = cacheManager
}

func (c *AppContext) GetCache() *caching.Manager {
	return c.cache
}

func (c *AppContext) SetLogger(logger *logging.Logger) {
	c.logger = logger
}

func (c *AppContext) GetLogger() *logging.Logger {
	return c.logger
}

func (c *AppContext) SetStdout(stdout io.Writer) {
	c.stdout = stdout
}

func (c *AppContext) Stdout() io.Writer {
	return c.stdout
}

func (c *AppContext) SetStderr(stderr io.Writer) {
	c.stderr = stderr
}

func (c *AppContext) Stderr() io.Writer {
	return c.stderr
}

func (c *AppContext) SetContext(ctx context.Context) {
	c.context = ctx
}

func (c *AppContext) GetContext() context.Context {
	return c.context
}

func (c *AppContext) SetSecret(secret []byte) {
	c.secret = secret
}

func (c *AppContext) GetSecret() []byte {
	return c.secret
}
