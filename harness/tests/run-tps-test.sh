REPOPATH=`pwd`/../..

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

# localnet wallet default password
export  DUSK_WALLET_PASS="password"

# Network executables
export DUSK_BLINDBID="$REPOPATH/bin/blindbid-avx2"
export DUSK_BLOCKCHAIN="$REPOPATH/bin/dusk"
export DUSK_SEEDER="$REPOPATH/bin/voucher"


# Increase open files OS limit
ulimit -n 20000

# View Variables
export | grep DUSK_

# Start test harness
go test -v --count=1 --test.timeout=0  -run  TestMeasureNetworkTPS  -args -enable


