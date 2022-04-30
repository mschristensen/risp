SHELL:=/bin/bash

cmd=risp
subcommand=
timeout=30s
long_timeout=1m
dir=./...
run=.
flags=
parallel=3


# include test env vars if the make target contains the string "test", otherwise use dev env vars
ifneq (true,$(NO_ENV_FILE))
ifneq (,$(findstring test,$(MAKECMDGOALS)))
include ./configs/test.env
export $(shell sed 's/=.*//' ./configs/test.env)
else ifeq (,$(findstring docker,$(MAKECMDGOALS))$(findstring docs,$(MAKECMDGOALS)))
include ./configs/dev.env
export $(shell sed 's/=.*//' ./configs/dev.env)
endif
endif


## help:				Shows help messages.
.PHONY: help
help:
	@sed -n 's/^##//p' $(MAKEFILE_LIST)


## clean:				Cleans up build artefacts.
.PHONY: clean
clean:
	@rm -rf bin
	@mkdir bin


## lint:				Runs linters.
.PHONY: lint
lint:
	go fmt ./...
	go vet ./...
	golangci-lint run --timeout 5m0s ./...


## docs:				Starts the Go documentation server.
.PHONY: docs
docs:
	godoc -v -http=:6060


## mocks:				Generate mocks in all packages.
.PHONY:
mocks:
	@go install github.com/vektra/mockery/v2@latest
	@go generate ./...


## proto:				Builds the proto files.
.PHONY: proto
proto:
	@cd ./api/proto && $(MAKE) all && cd ../..


## build_dependencies:		Builds the application dependencies.
.PHONY: build_dependencies
build_dependencies:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0
	@mkdir -p ./bin


## build:				Builds the application. [cmd]
.PHONY: build
build: build_dependencies lint
	@go build -a -installsuffix cgo -ldflags="-w -s" -o ./bin/$(cmd) ./cmd/$(cmd)/...
	@go mod tidy


## test_integ_deps:		Prepares dependencies for running integration tests, creating and starting all containers.
.PHONY: test_integ_deps
test_integ_deps: build_dependencies


## test_start_containers:		Starts all the existing containers for the test environment.
.PHONY: test_start_containers
test_start_containers:
	echo "Nothing to do"


## test_integ:			Runs integration tests. [timeout, dir, flags, run]
.PHONY: test_integ
test_integ: #test_start_containers
	@echo "Running integration tests on $(run)"
	@go test $(flags) -parallel $(parallel) -failfast -timeout $(long_timeout) $(dir) -run $(run)


## test_unit:			Runs unit tests. [run, flags, timeout, dir]
.PHONY: test_unit
test_unit: build_dependencies
	@echo "Running tests on $(run)"
	@go test $(flags) -short -parallel $(parallel) -failfast -timeout $(timeout) $(dir) -run $(run)


## exec:				Executes the built application binary. [cmd, subcommand, flags]
.PHONY: exec
exec:
	@./bin/$(cmd) $(subcommand) $(flags)


## run:				Runs the application using go run. [cmd, subcommand, flags]
.PHONY: run
run:
	@go run ./cmd/$(cmd) $(subcommand) $(flags)
