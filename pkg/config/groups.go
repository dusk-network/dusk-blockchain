// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package config

type generalConfiguration struct {
	Network string
}

type timeoutConfiguration struct {
	TimeoutGetMempoolTXsBySize int64
	TimeoutGetMempoolTXs       int64
}

type loggerConfiguration struct {
	Level  string
	Output string
	Format string
}

type networkConfiguration struct {
	MaxDupeMapItems  uint32
	MaxDupeMapExpire uint32

	ServiceFlag uint8
}

type clientConfiguration struct {
	Network     string
	Address     string
	DialTimeout int
}

type kadcastConfiguration struct {
	Enabled       bool
	Address       string
	BootstrapAddr []string

	Grpc clientConfiguration
}

// pkg/core/database package configs.
type databaseConfiguration struct {
	Driver string
	Dir    string
}

// pprof configs.
// See also utils/diagnostics/ProfileSet.
type profileConfiguration struct {
	Name     string
	Interval uint
	Duration uint
	Start    bool
}

// pkg/rpc package configs.
type rpcConfiguration struct {
	Network             string
	Address             string
	SessionDurationMins uint
	RequireSession      bool

	EnableTLS bool
	CertFile  string
	KeyFile   string

	User string
	Pass string

	Rusk ruskConfiguration
}

// rpc/rusk related configurations.
type ruskConfiguration struct {
	Network string
	Address string

	// timeout for rusk calls.
	ContractTimeout   uint
	DefaultTimeout    uint
	ConnectionTimeout uint
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

type apiConfiguration struct {
	Enabled        bool
	Address        string
	EnableTLS      bool
	CertFile       string
	KeyFile        string
	DBFile         string
	ExpirationTime int
}

type notificationConfiguration struct {
	BrokersNum       uint
	ClientsPerBroker uint
}

// Performance parameters.
type performanceConfiguration struct {
	AccumulatorWorkers int
}

type mempoolConfiguration struct {
	MaxSizeMB uint32
	PoolType  string

	MaxInvItems uint32

	PropagateTimeout string
	PropagateBurst   uint32

	// diskpool config
	DiskPoolDir string

	// Hashmap config
	HashMapPreallocTxs uint32

	// Enables mempool updates at startup
	Updates updates

	ExtractionDelaySecs uint32
}

type updates struct {
	// NumNodes is count of the nodes to ask for mempool transactions at start-up
	NumNodes uint8

	// Disabled is false by default which enables updates with default NumNodes
	Disabled bool
}

type consensusConfiguration struct {
	// Path to a file that stores Consensus Keys / BLS public and secret keys
	// if file does not exist, it will be created at startup.
	KeysFile string

	// ConsensusTimeOut is the time out for consensus step timers.
	ConsensusTimeOut int64
	// UseCompressedKeys determines if AggregatePks works with compressed or uncompressed pks.
	UseCompressedKeys bool

	// ThrottleMilli determines number of Milliseconds to throttle block
	// acceptance if Consensus time is less than config.ConsensusTimeThreshold.
	ThrottleMilli int64

	// ThrottleIterMilli determines number of Milliseconds to throttle VerifyST.
	ThrottleIterMilli int64
}

type stateConfiguration struct {
	// PersistEvery N blocks the state in rusk
	PersistEvery  uint64
	BlockGasLimit uint64
}
