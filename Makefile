BINDIR := $(CURDIR)/bin

# go option
PKG       := ./..
TAGS      :=
TESTS     := .
TESTFLAGS :=
LDFLAGS   := -w -s
GOFLAGS   :=

# git info
GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

.PHONY: all
all: build

# ---------------------------------------------------------------
# build

.PHONY: build
build: $(BINDIR)/flv-parse

$(BINDIR)/flv-parse: main.go flv/flv.go
	GO111MODULE=on go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $@ main.go


# ---------------------------------------------------------------
# clean

.PHONY: clean
clean:
	@rm -f ${BINDIR}/*

# ---------------------------------------------------------------
# info

.PHONY: info
info:
	@echo "Version:        ${VERSION}"
	@echo "Git Tag:        ${GIT_TAG}"
	@echo "Git Commit:     ${GIT_COMMIT}"
	@echo "Git Tree State: ${GIT_DIRTY}"