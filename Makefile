
PROJECT_NAME := "dusk-go"
PKG := "gitlab.dusk.network/dusk-core/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
TEST_LIST := $(shell go list ${PKG}/...)
#TEST_FLAGS := "-count=1"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
.PHONY: all dep build clean test coverage coverhtml lint
all: build
lint: ## Lint the files
	@golint -set_exit_status ${PKG_LIST}
test: ## Run unittests
	@go test  -p 1 -short ${TEST_LIST}
race: dep ## Run data race detector
	@go test ${TEST_FLAGS} -race -v ${TEST_LIST}
coverage: ## Generate global code coverage report
	chmod u+x coverage.sh
	./coverage.sh;
coverhtml: ## Generate global code coverage report in HTML
	chmod u+x coverage.sh
	./coverage.sh html;
dep: ## Get the dependencies
	@go get -v -d ./...
	# @go get -u github.com/golang/lint/golint
build: dep ## Build the binary file
	@go build -i -v $(PKG)
clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)
	@go clean -testcache
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
