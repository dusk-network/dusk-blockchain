
PROJECT_NAME := "dusk-blockchain"
PKG := "github.com/dusk-network/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
#TEST_FLAGS := "-count=1"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
.PHONY: all dep build clean test coverage coverhtml lint
all: build
lint: ## Lint the files
	GOBIN=$(PWD)/bin go run scripts/build.go lint
test: ## Run unittests
	@go test $(TFLAGS) -p 1 -short ${PKG_LIST}
test-harness: ## Run harness tests
	@go test -v --count=1 --test.timeout=0 ./harness/tests/... -args -enable
get-blindbid: ## download dusk-blindbidproof
	@rm -rf ${PWD}/bin/blindbid-linux-amd64 || true
	@wget -P ${PWD}/bin/ https://github.com/dusk-network/dusk-blindbidproof/releases/download/v0.1.0/blindbid-linux-amd64 && chmod +x ${PWD}/bin/blindbid-linux-amd64
test-harness-ci: get-blindbid build
	NETWORK_SIZE=7 DUSK_BLOCKCHAIN=${PWD}/bin/dusk DUSK_BLINDBID=${PWD}/bin/blindbid-linux-amd64 DUSK_SEEDER=${PWD}/bin/voucher DUSK_WALLET_PASS="password" make test-harness
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
netcollector: build
	./bin/netcollector
###################################CROSS#################################################
install-tools:
	go get -u github.com/karalabe/xgo
cross: \
	dusk-linux dusk-linux-arm dusk-darwin dusk-windows \
	voucher-linux voucher-linux-arm voucher-darwin voucher-windows \
	wallet-linux wallet-linux-arm wallet-darwin wallet-windows \
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
