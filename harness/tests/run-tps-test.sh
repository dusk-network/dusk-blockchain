# Run setup-env script before this one

# Config template to be loaded
# default - enables default gossip configiration
# kadcast - enables Kadcast over RCUDP
export DUSK_PROFILE=$1

# Test harness runnning 300 nodes
export DUSK_NETWORK_SIZE=300

# Test harness consensus-running nodes
# Must be less than $NETWORK_SIZE and not more than provided wallet.dat files
export DUSK_CONSENSUS_NODES=10

# Enable TestMeasureNetworkTPS and Data collecting
export DUSK_ENABLE_TPS_TEST=1

# Increase open files OS limit
ulimit -n 20000

# View Variables
export | grep DUSK_

# Start test harness
go test -v --count=1 --test.timeout=0  -run  TestMeasureNetworkTPS  -args -enable


