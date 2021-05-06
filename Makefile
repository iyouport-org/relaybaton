PROJECT="relaybaton"

# Constants
MK_FILE_PATH = $(lastword $(MAKEFILE_LIST))
PRJ_DIR      = $(abspath $(dir $(MK_FILE_PATH)))
GOPATH_ENV  ?= $(shell go env GOPATH)
GOROOT = $(GOPATH_ENV)/src/github.com/cloudflare/go
ifeq ($(OS),Windows_NT)
	MAKESCRIPT := make.bat
else
	MAKESCRIPT := make.bash
endif

###############
#
# Build targets
#
##############################

core: $(GOROOT)/pkg $(PRJ_DIR)/vendor
	GOROOT=$(GOROOT) $(GOROOT)/bin/go build -o $(PRJ_DIR)/bin/relaybaton $(PRJ_DIR)/cmd/cli/main.go

###############
#
# Build GOROOT
#
##############################

$(PRJ_DIR)/vendor:
	go mod vendor

$(GOROOT)/pkg: $(GOROOT)
	cd $(GOPATH_ENV)/src/github.com/cloudflare/go/src/ && \
	./$(MAKESCRIPT) && \
	cd $(PRJ_DIR)

$(GOROOT):
	git clone https://github.com/cloudflare/go.git $(GOPATH_ENV)/src/github.com/cloudflare/go
