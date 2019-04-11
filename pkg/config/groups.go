package config

type GeneralConfiguration struct {
	Network string
}

type LoggerConfiguration struct {
	Level string
}

type NetworkConfiguration struct {
	Address string
}

// pkg/core/database package configs
type DatabaseConfiguration struct {
	DriverName string
	Path       string
}

// pprof configs
type ProfileConfiguration struct {
	Address string

	// enables CPU profiling for the current process.
	// While profiling, the profile will be buffered and written to CPUProf file.
	CpuFile string
	// Write mem profile to the specified file
	MemFile string
}

// pkg/rpc package configs
type RPCServerConfiguration struct {
	Address string
}
