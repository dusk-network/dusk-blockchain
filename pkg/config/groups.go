package config

type generalConfiguration struct {
	Network    string
	LightNode bool
}

type loggerConfiguration struct {
	Level   string
	Output  string
	Monitor logMonitorConfiguration
}

// log based monitoring defined in pkg/eventmon/logger
type logMonitorConfiguration struct {
	Enabled      bool
	Target       string
	StreamErrors bool
}

type networkConfiguration struct {
	Seeder  seedersConfiguration
	Monitor monitorConfiguration
	Port    string
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
type profConfiguration struct {
	Address string

	// enables CPU profiling for the current process.
	// While profiling, the profile will be buffered and written to CPUProf file.
	CPUFile string
	// Write mem profile to the specified file
	MemFile string
}

// pkg/rpc package configs
type rpcConfiguration struct {
	Network string
	Address string
	// Enabled bool
	// User    string
	// Pass    string
}

type gqlConfiguration struct {
	Enabled bool
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
