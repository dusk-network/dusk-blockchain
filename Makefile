
PROJECT_NAME := "dusk-blockchain"
PKG := "github.com/dusk-network/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
#TEST_FLAGS := "-count=1"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
.PHONY: all dep build clean test coverage coverhtml lint
all: build
lint: ## Lint the files
	GOBIN=$(PWD)/bin go run scripts/build.go lint
go-analyzer: ## Run go-analyzer
	GOBIN=$(PWD)/bin go run scripts/build.go go-analyzer
test: ## Run unittests
	@go test $(TFLAGS) -short ${PKG_LIST}
test-harness-unit: build ## Run a specified harness unit test e.g make UNIT_TEST=TestForgedBlock test-harness-unit
	MOCK_ADDRESS=127.0.0.1:8080 DUSK_NETWORK_SIZE=9 DUSK_BLOCKCHAIN=${PWD}/bin/dusk DUSK_UTILS=${PWD}/bin/utils DUSK_SEEDER=${PWD}/bin/voucher DUSK_WALLET_PASS="password" \
	go test -v --count=1 --test.timeout=0 ./harness/tests/ -run $(UNIT_TEST) -args -enable
test-harness: ## Run harness tests
	@go test -v --count=1 --test.timeout=0 ./harness/tests/ -args -enable
test-harness-ci: build
	MOCK_ADDRESS=127.0.0.1:8080 DUSK_NETWORK_SIZE=3 DUSK_BLOCKCHAIN=${PWD}/bin/dusk DUSK_UTILS=${PWD}/bin/utils DUSK_SEEDER=${PWD}/bin/voucher DUSK_WALLET_PASS="password" make test-harness
test-harness-alive: stop build
	MOCK_ADDRESS=127.0.0.1:9191 DUSK_NETWORK_SIZE=9 DUSK_BLOCKCHAIN=${PWD}/bin/dusk DUSK_UTILS=${PWD}/bin/utils DUSK_SEEDER=${PWD}/bin/voucher DUSK_WALLET_PASS="password" \
	go test -v --count=1 --test.timeout=0 ./harness/tests/ -run TestMultipleBiddersProvisioners -args -enable -keepalive
test-harness-tps: stop build
	MOCK_ADDRESS=127.0.0.1:9191 DUSK_NETWORK_SIZE=9 DUSK_BLOCKCHAIN=${PWD}/bin/dusk DUSK_UTILS=${PWD}/bin/utils DUSK_SEEDER=${PWD}/bin/voucher DUSK_WALLET_PASS="password" \
	go test -v --count=1 --test.timeout=0 ./harness/tests/ -run TestMeasureNetworkTPS -args -enable -keepalive
test-harness-session:
	REQUIRE_SESSION=true make test-harness-alive
test-harness-race-alive: stop build-race
	MOCK_ADDRESS=127.0.0.1:9191 DUSK_NETWORK_SIZE=9 DUSK_BLOCKCHAIN=${PWD}/bin/dusk DUSK_UTILS=${PWD}/bin/utils DUSK_SEEDER=${PWD}/bin/voucher DUSK_WALLET_PASS="password" \
	go test -v --count=1 --test.timeout=0 ./harness/tests/ -run TestMultipleBiddersProvisioners  -args -enable -keepalive
test-harness-race-debug-alive: stop build-race-debug
	MOCK_ADDRESS=127.0.0.1:9191 DUSK_NETWORK_SIZE=9 DUSK_BLOCKCHAIN=${PWD}/bin/dusk DUSK_UTILS=${PWD}/bin/utils DUSK_SEEDER=${PWD}/bin/voucher DUSK_WALLET_PASS="password" \
	go test -v --count=1 --test.timeout=0 ./harness/tests/ -run TestMultipleBiddersProvisioners  -args -enable -keepalive
race: dep ## Run data race detector
	@go test $(TFLAGS) -race -v ${PKG_LIST}
coverage: ## Generate global code coverage report
	chmod u+x coverage.sh
	./coverage.sh;
coverhtml: ## Generate global code coverage report in HTML
	chmod u+x coverage.sh
	./coverage.sh html;
dep: ## Get the dependencies
	go mod download
build: dep ## Build the binary file
	GOBIN=$(PWD)/bin go run scripts/build.go install
build-race: dep ## Build the binary file
	GOBIN=$(PWD)/bin go run scripts/build.go install -race
build-race-debug: dep ## Build the binary file
	GOBIN=$(PWD)/bin go run scripts/build.go install -race -debug
clean: ## Remove previous build
	@rm -f ./bin
	@go clean -testcache
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
dusk: build
	./bin/dusk --config=dusk.toml
voucher: build
	./bin/voucher
wallet: build
	./bin/wallet
mock: build
	./bin/utils mock --grpcmockhost=127.0.0.1:9191
mockrusk: build
	./bin/utils mockrusk --rusknetwork=tcp --ruskaddress=127.0.0.1:10000 \
	--walletstore=/tmp/localnet-137601832/node-9003/walletDB/ \
	--walletfile=./harness/data/wallet-9000.dat
devnet: stop
	./devnet.sh
netcollector: build
	./bin/netcollector
stop:
	echo "will stop dusk app"
	killall dusk || true
	killall voucher || true
	killall utils || true
	killall filebeat || true
###################################CROSS#################################################
install-tools:
	go get -u github.com/karalabe/xgo
cross: \
	dusk-linux dusk-linux-arm dusk-darwin dusk-windows \
	voucher-linux voucher-linux-arm voucher-darwin voucher-windows \
	wallet-linux wallet-linux-arm wallet-darwin wallet-windows
###################################DUSK#################################################
dusk-linux: install-tools
	xgo --go=latest --targets=linux/amd64 -out=./bin/dusk ./cmd/dusk
	xgo --go=latest --targets=linux/386 -out=./bin/dusk  ./cmd/dusk
dusk-linux-arm: install-tools
	xgo --go=latest --targets=linux/arm-5 -out=./bin/dusk ./cmd/dusk
	xgo --go=latest --targets=linux/arm-6 -out=./bin/dusk ./cmd/dusk
	xgo --go=latest --targets=linux/arm-7 -out=./bin/dusk ./cmd/dusk
	xgo --go=latest --targets=linux/arm64 -out=./bin/dusk ./cmd/dusk
dusk-darwin: install-tools
	xgo --go=latest --targets=darwin/amd64 -out=./bin/dusk ./cmd/dusk
	xgo --go=latest --targets=darwin/386 -out=./bin/dusk ./cmd/dusk
dusk-windows: install-tools
	#xgo --go=latest --targets=windows/amd64 -out=./bin/dusk ./cmd/dusk #407
	#xgo --go=latest --targets=windows/386 -out=./bin/dusk ./cmd/dusk #407
###################################VOUCHER##############################################
voucher-linux: install-tools
	xgo --go=latest --targets=linux/amd64 -out=./bin/voucher ./cmd/voucher
	xgo --go=latest --targets=linux/386 -out=./bin/voucher  ./cmd/voucher
voucher-linux-arm: install-tools
	xgo --go=latest --targets=linux/arm-5 -out=./bin/voucher ./cmd/voucher
	xgo --go=latest --targets=linux/arm-6 -out=./bin/voucher ./cmd/voucher
	xgo --go=latest --targets=linux/arm-7 -out=./bin/voucher ./cmd/voucher
	xgo --go=latest --targets=linux/arm64 -out=./bin/voucher ./cmd/voucher
voucher-darwin: install-tools
	xgo --go=latest --targets=darwin/amd64 -out=./bin/voucher ./cmd/voucher
	xgo --go=latest --targets=darwin/386 -out=./bin/voucher ./cmd/voucher
voucher-windows: install-tools
	xgo --go=latest --targets=windows/amd64 -out=./bin/voucher ./cmd/voucher
	xgo --go=latest --targets=windows/386 -out=./bin/voucher ./cmd/voucher
###################################WALLET###############################################
wallet-linux: install-tools
	xgo --go=latest --targets=linux/amd64 -out=./bin/wallet ./cmd/wallet
	xgo --go=latest --targets=linux/386 -out=./bin/wallet  ./cmd/wallet
wallet-linux-arm: install-tools
	xgo --go=latest --targets=linux/arm-5 -out=./bin/wallet ./cmd/wallet
	xgo --go=latest --targets=linux/arm-6 -out=./bin/wallet ./cmd/wallet
	xgo --go=latest --targets=linux/arm-7 -out=./bin/wallet ./cmd/wallet
	xgo --go=latest --targets=linux/arm64 -out=./bin/wallet ./cmd/wallet
wallet-darwin: install-tools
	xgo --go=latest --targets=darwin/amd64 -out=./bin/wallet ./cmd/wallet
	xgo --go=latest --targets=darwin/386 -out=./bin/wallet ./cmd/wallet
wallet-windows: install-tools
	#xgo --go=latest --targets=windows/amd64 -out=./bin/wallet ./cmd/wallet #407
	#xgo --go=latest --targets=windows/386 -out=./bin/wallet ./cmd/wallet #407
########################################################################################
