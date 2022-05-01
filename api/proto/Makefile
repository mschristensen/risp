SHELL:=/bin/bash

define COMPILE_CMD
docker run -v ${PWD}:/defs namely/protoc-all -f ./risp.proto -l go
endef


.PHONY: help
help: ## Shows this help
	@grep -E '^[0-9a-zA-Z_-]+:(.*?## .*)?$$' $(MAKEFILE_LIST) | sed 's/^Makefile://' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: all
all: dependencies protobuf ## Runs everything


.PHONY: dependencies
dependencies: ## Creates the directories containing the compiled go libraries
	@docker pull namely/protoc-all


.PHONY: protobuf
protobuf: ## Generates the go libraries.
	@$(COMPILE_CMD)


## mocks:				Generate mocks in all packages.
.PHONY:
mocks:
	@mockery --dir ./gen/pb-go/github.com/mschristensen/risp/api/build/go/ --output ./gen/pb-go/github.com/mschristensen/risp/api/build/go/mocks --all
