export DUSK_BLOCKCHAIN=/home/jules/go/src/github.com/dusk-network/dusk-blockchain/launch/testnet/testnet
export DUSK_BLINDBID=/home/jules/go/src/github.com/dusk-network/dusk-blockchain/launch/testnet/blindbid-avx2
export DUSK_SEEDER=/home/jules/go/src/gitlab.dusk.network/dusk-core/dusk-voucher-seeder/dusk-voucher-seeder
export DUSK_WALLET_PASS=password

cd launch/testnet
go build 
cd ../..
