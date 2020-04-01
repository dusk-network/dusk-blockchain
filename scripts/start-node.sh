go build

# Launch processes in the background, and ignore their output
blindbid &>/dev/null &
../bin/testnet &>/dev/null &
