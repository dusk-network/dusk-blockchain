package config

type generalConfiguration struct {
	Network string
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
	Port    string
	Enabled bool
	User    string
	Pass    string
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
