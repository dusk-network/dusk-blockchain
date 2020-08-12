package config

type generalConfiguration struct {
	Network    string
	WalletOnly bool
}

type loggerConfiguration struct {
	Level   string
	Output  string
	Monitor logMonitorConfiguration
}

// log based monitoring defined in pkg/eventmon/logger
type logMonitorConfiguration struct {
	Enabled      bool
	Rpc          string //nolint
	Transport    string
	Address      string
	StreamErrors bool
}

type networkConfiguration struct {
	Seeder  seedersConfiguration
	Monitor monitorConfiguration
	Port    string
}

type kadcastConfiguration struct {
	Enabled bool
	Network string

	// IP nature
	Address string

	// a set of network addresses of bootstrapping nodes
	Bootstrappers []string

	// Kadcast protocol configs
	MaxDelegatesNum byte

	Raptor bool
}

type monitorConfiguration struct {
	Address string
	Enabled bool
}

type seedersConfiguration struct {
	Addresses []string
	Fixed     []string
}

// pkg/core/database package configs
type databaseConfiguration struct {
	Driver string
	Dir    string
}

// wallet configs
type walletConfiguration struct {
	File  string
	Store string
}

// pprof configs
// See also utils/diagnostics/ProfileSet
type profileConfiguration struct {
	Name     string
	Interval uint
	Duration uint
	Start    bool
}

// pkg/rpc package configs
type rpcConfiguration struct {
	Network string
	Address string

	EnableTLS bool
	CertFile  string
	KeyFile   string

	User string
	Pass string
}

type gqlConfiguration struct {
	// TODO: Keep 'Enabled' option?
	Enabled bool
	Network string
	Address string

	EnableTLS bool
	CertFile  string
	KeyFile   string

	MaxRequestLimit uint

	Notification notificationConfiguration
}

type notificationConfiguration struct {
	BrokersNum       uint
	ClientsPerBroker uint
}

// Performance parameters
type performanceConfiguration struct {
	AccumulatorWorkers int
}

type mempoolConfiguration struct {
	MaxSizeMB   uint32
	PoolType    string
	PreallocTxs uint32
	MaxInvItems uint32
}

type consensusConfiguration struct {
	DefaultLockTime uint64
	DefaultAmount   uint64
}
