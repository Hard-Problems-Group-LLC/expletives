SHELL := /bin/sh
.DEFAULT_GOAL := all

GO ?= go
MODULE := github.com/Hard-Problems-Group-LLC/expletives
COMMANDS := expletives-test expletivesctl

# Do not permit a command-line or environment override to broaden clean's
# deletion boundary.
override PROJECT_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
override BUILD_ROOT := $(PROJECT_ROOT)/build

DEBUG_ARTIFACTS := $(addprefix $(BUILD_ROOT)/debug/,$(COMMANDS))
RELEASE_ARTIFACTS := $(addprefix $(BUILD_ROOT)/release/,$(COMMANDS))
PROFILING_ARTIFACTS := $(addprefix $(BUILD_ROOT)/profiling/,$(COMMANDS))

COMMON_BUILD_FLAGS := -mod=readonly -buildvcs=true
BUILDINFO_PATH := $(MODULE)/internal/buildinfo.buildMode

.PHONY: all build release profiling
.PHONY: check-debug-artifacts check-release-artifacts
.PHONY: check-profiling-artifacts
.PHONY: clean test test-unit test-integration test-race
.PHONY: fmt-check vet verify smoke FORCE

all: build release profiling

build: check-debug-artifacts

release: check-release-artifacts

profiling: check-profiling-artifacts

$(BUILD_ROOT)/debug/%: FORCE
	@mkdir -p "$(@D)"
	CGO_ENABLED=0 $(GO) build $(COMMON_BUILD_FLAGS) \
		-gcflags='all=-N -l' \
		-ldflags='-X=$(BUILDINFO_PATH)=debug' \
		-o "$@" "./cmd/$*"

$(BUILD_ROOT)/release/%: FORCE
	@mkdir -p "$(@D)"
	CGO_ENABLED=0 $(GO) build $(COMMON_BUILD_FLAGS) -trimpath \
		-ldflags='-X=$(BUILDINFO_PATH)=release' \
		-o "$@" "./cmd/$*"

$(BUILD_ROOT)/profiling/%: FORCE
	@mkdir -p "$(@D)"
	CGO_ENABLED=0 $(GO) build $(COMMON_BUILD_FLAGS) -trimpath \
		-ldflags='-X=$(BUILDINFO_PATH)=profiling' \
		-o "$@" "./cmd/$*"

check-debug-artifacts: $(DEBUG_ARTIFACTS)
	@test -x "$(BUILD_ROOT)/debug/expletives-test"
	@test -x "$(BUILD_ROOT)/debug/expletivesctl"

check-release-artifacts: $(RELEASE_ARTIFACTS)
	@test -x "$(BUILD_ROOT)/release/expletives-test"
	@test -x "$(BUILD_ROOT)/release/expletivesctl"

check-profiling-artifacts: $(PROFILING_ARTIFACTS)
	@test -x "$(BUILD_ROOT)/profiling/expletives-test"
	@test -x "$(BUILD_ROOT)/profiling/expletivesctl"

clean:
	@test "$(BUILD_ROOT)" = "$(PROJECT_ROOT)/build"
	@rm -rf -- "$(BUILD_ROOT)"

test: test-unit test-integration

test-unit:
	$(GO) test -mod=readonly ./...

test-integration: check-debug-artifacts
	$(GO) test -mod=readonly -tags=integration \
		-run '^TestDebugBinaryPTYProcessLifecycle$$' \
		./cmd/expletives-test

# The shipped binaries remain CGO-free. Go's race runtime itself requires
# cgo on supported platforms, so cgo is enabled only for this verification.
test-race:
	CGO_ENABLED=1 $(GO) test -mod=readonly -race ./...

fmt-check:
	@unformatted="$$(find . -type f -name '*.go' \
		-not -path './FieldManual/*' \
		-not -path './.codex-home/*' \
		-not -path './.local/*' \
		-not -path './build/*' -print | sort | xargs -r gofmt -l)"; \
	if test -n "$$unformatted"; then \
		echo "Go files require gofmt:" >&2; \
		echo "$$unformatted" >&2; \
		exit 1; \
	fi

vet:
	$(GO) vet -mod=readonly ./...

smoke: all
	@set -e; for mode in debug release profiling; do \
		"$(BUILD_ROOT)/$$mode/expletives-test" --version >/dev/null; \
		"$(BUILD_ROOT)/$$mode/expletives-test" --self-check >/dev/null; \
		"$(BUILD_ROOT)/$$mode/expletivesctl" --version >/dev/null; \
		"$(BUILD_ROOT)/$$mode/expletivesctl" --help >/dev/null 2>&1; \
	done

verify: fmt-check vet test test-race all smoke

FORCE:
