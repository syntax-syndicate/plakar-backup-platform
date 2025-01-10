package appcontext

import (
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/encryption/keypair"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/google/uuid"
)

type AppContext struct {
	events *events.Receiver
	cache  *caching.Manager
	logger *logging.Logger

	numCPU      int
	username    string
	homeDir     string
	hostname    string
	commandLine string
	machineID   string
	keyFromFile string
	cacheDir    string
	keyringDir  string

	operatingSystem string
	architecture    string
	processID       int

	plakarClient string

	cwd string

	maxConcurrency int

	identity uuid.UUID
	keypair  *keypair.KeyPair
}

func NewAppContext() *AppContext {
	return &AppContext{
		events: events.New(),
	}
}

func (c *AppContext) Close() {
	c.events.Close()
}

func (c *AppContext) Events() *events.Receiver {
	return c.events
}

func (c *AppContext) SetCWD(cwd string) {
	c.cwd = cwd
}

func (c *AppContext) GetCWD() string {
	return c.cwd
}

func (c *AppContext) SetNumCPU(numCPU int) {
	c.numCPU = numCPU
}

func (c *AppContext) GetNumCPU() int {
	return c.numCPU
}

func (c *AppContext) SetUsername(username string) {
	c.username = username
}

func (c *AppContext) GetUsername() string {
	return c.username
}

func (c *AppContext) SetHostname(hostname string) {
	c.hostname = hostname
}

func (c *AppContext) GetHostname() string {
	return c.hostname
}

func (c *AppContext) SetCommandLine(commandLine string) {
	c.commandLine = commandLine
}

func (c *AppContext) GetCommandLine() string {
	return c.commandLine
}

func (c *AppContext) SetMachineID(machineID string) {
	c.machineID = machineID
}

func (c *AppContext) GetMachineID() string {
	return c.machineID
}

func (c *AppContext) SetKeyFromFile(keyFromFile string) {
	c.keyFromFile = keyFromFile
}

func (c *AppContext) GetKeyFromFile() string {
	return c.keyFromFile
}

func (c *AppContext) SetHomeDir(homeDir string) {
	c.homeDir = homeDir
}

func (c *AppContext) GetHomeDir() string {
	return c.homeDir
}

func (c *AppContext) SetCacheDir(cacheDir string) {
	c.cacheDir = cacheDir
}

func (c *AppContext) GetCacheDir() string {
	return c.cacheDir
}

func (c *AppContext) SetOperatingSystem(operatingSystem string) {
	c.operatingSystem = operatingSystem
}

func (c *AppContext) GetOperatingSystem() string {
	return c.operatingSystem
}

func (c *AppContext) SetArchitecture(architecture string) {
	c.architecture = architecture
}

func (c *AppContext) GetArchitecture() string {
	return c.architecture
}

func (c *AppContext) SetProcessID(processID int) {
	c.processID = processID
}

func (c *AppContext) GetProcessID() int {
	return c.processID
}

func (c *AppContext) SetKeyringDir(keyringDir string) {
	c.keyringDir = keyringDir
}

func (c *AppContext) GetKeyringDir() string {
	return c.keyringDir
}

func (c *AppContext) SetIdentity(identity uuid.UUID) {
	c.identity = identity
}

func (c *AppContext) GetIdentity() uuid.UUID {
	return c.identity
}

func (c *AppContext) SetKeypair(keypair *keypair.KeyPair) {
	c.keypair = keypair
}

func (c *AppContext) GetKeypair() *keypair.KeyPair {
	return c.keypair
}

func (c *AppContext) SetPlakarClient(plakarClient string) {
	c.plakarClient = plakarClient
}

func (c *AppContext) GetPlakarClient() string {
	return c.plakarClient
}

func (c *AppContext) SetMaxConcurrency(maxConcurrency int) {
	c.maxConcurrency = maxConcurrency
}

func (c *AppContext) GetMaxConcurrency() int {
	return c.maxConcurrency
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
