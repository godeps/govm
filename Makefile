SHELL := /usr/bin/env bash

PROFILE ?= release
BOXLITE_DIR ?=
BOXLITE_RUNTIME_DIR ?=
GO_TEST_FLAGS ?= -v

CURRENT_PLATFORM := $(shell go env GOOS)_$(shell go env GOARCH)
CURRENT_NATIVE_DIR := internal/native/$(CURRENT_PLATFORM)
CURRENT_BRIDGE_FILE := $(CURRENT_NATIVE_DIR)/libgovm_boxlite_bridge.a
BRIDGE_BUILD_FILE_RELEASE := rust-bridge/target/release/libgovm_boxlite_bridge.a
BRIDGE_BUILD_FILE_DEBUG := rust-bridge/target/debug/libgovm_boxlite_bridge.a

.PHONY: help fmt test-stub build-stub doctor gc platform-check
.PHONY: doctor-json
.PHONY: bridge bridge-install-local bridge-install-local-release bridge-install-local-debug
.PHONY: native-sync native-verify native-verify-current native-prepare
.PHONY: runtime-sync runtime-verify runtime-prepare
.PHONY: test-native build-native validate validate-native bootstrap clean

help:
	@echo "govm automation targets"
	@echo ""
	@echo "Core:"
	@echo "  make fmt                          # gofmt all Go files"
	@echo "  make test-stub                    # run tests in stub mode (default)"
	@echo "  make build-stub                   # build in stub mode (default)"
	@echo "  make validate                     # fmt + test-stub + build-stub"
	@echo "  make doctor-json                 # machine-readable doctor output"
	@echo "  make gc                           # remove stopped boxes (use GC_ARGS='--all --force' etc.)"
	@echo ""
	@echo "Native bridge:"
	@echo "  make bridge PROFILE=release|debug # build rust static bridge"
	@echo "  make bridge-install-local         # build bridge and install into current platform native dir"
	@echo "  make native-sync BOXLITE_DIR=...  # sync prebuilt native artifacts from local boxlite"
	@echo "  make native-verify                # verify all platform native bridge artifacts"
	@echo "  make native-verify-current        # verify current platform bridge artifact"
	@echo "  make test-native                  # run native tests (-tags govm_native)"
	@echo "  make validate-native              # native-prepare + test-native + build-native"
	@echo "  make runtime-sync BOXLITE_RUNTIME_DIR=... # sync runtime binaries for current platform"
	@echo "  make runtime-verify               # verify runtime binaries in repo"
	@echo "  make platform-check               # check expected multi-platform assets"
	@echo ""
	@echo "One-shot bootstrap:"
	@echo "  make bootstrap BOXLITE_DIR=... BOXLITE_RUNTIME_DIR=... # sync native/runtime + native tests/build"

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

test-stub:
	go test $(GO_TEST_FLAGS) ./...

build-stub:
	go build ./...

doctor:
	go run ./cmd/govm-doctor

doctor-json:
	go run ./cmd/govm-doctor --json

gc:
	go run ./cmd/govm-gc $(GC_ARGS)

platform-check:
	./scripts/verify-platform-assets.sh

bridge:
	./scripts/build-bridge.sh $(PROFILE)

bridge-install-local-release: PROFILE := release
bridge-install-local-release: bridge
	@mkdir -p "$(CURRENT_NATIVE_DIR)"
	cp "$(BRIDGE_BUILD_FILE_RELEASE)" "$(CURRENT_BRIDGE_FILE)"
	@echo "installed $(CURRENT_BRIDGE_FILE)"

bridge-install-local-debug: PROFILE := debug
bridge-install-local-debug: bridge
	@mkdir -p "$(CURRENT_NATIVE_DIR)"
	cp "$(BRIDGE_BUILD_FILE_DEBUG)" "$(CURRENT_BRIDGE_FILE)"
	@echo "installed $(CURRENT_BRIDGE_FILE)"

bridge-install-local: bridge-install-local-release

native-sync:
	@if [[ -z "$(BOXLITE_DIR)" ]]; then \
		echo "BOXLITE_DIR is required, e.g. make native-sync BOXLITE_DIR=../boxlite"; \
		exit 1; \
	fi
	./scripts/vendor-boxlite.sh "$(BOXLITE_DIR)"

runtime-sync:
	@if [[ -z "$(BOXLITE_RUNTIME_DIR)" ]]; then \
		echo "BOXLITE_RUNTIME_DIR is required, e.g. make runtime-sync BOXLITE_RUNTIME_DIR=/tmp/boxlite-runtime"; \
		exit 1; \
	fi
	./scripts/vendor-runtime.sh "$(BOXLITE_RUNTIME_DIR)" "$(CURRENT_PLATFORM)"

runtime-verify:
	./scripts/verify-runtime.sh "$(CURRENT_PLATFORM)"

runtime-prepare: runtime-verify

native-verify:
	./scripts/verify-native.sh

native-verify-current:
	@if [[ ! -f "$(CURRENT_BRIDGE_FILE)" ]]; then \
		echo "missing: $(CURRENT_BRIDGE_FILE)"; \
		echo "run: make native-sync BOXLITE_DIR=... or make bridge-install-local"; \
		exit 1; \
	fi
	@echo "found: $(CURRENT_BRIDGE_FILE)"

native-prepare:
	@if [[ ! -f "$(CURRENT_BRIDGE_FILE)" ]]; then \
		echo "bridge missing for $(CURRENT_PLATFORM), building local bridge..."; \
		$(MAKE) bridge-install-local-release; \
	fi
	@$(MAKE) native-verify-current

test-native: native-prepare runtime-prepare
	go test $(GO_TEST_FLAGS) -tags govm_native ./...

build-native: native-prepare runtime-prepare
	go build -tags govm_native ./...

validate: fmt test-stub build-stub

validate-native: native-prepare test-native build-native

bootstrap: native-sync runtime-sync native-prepare runtime-prepare validate-native

clean:
	rm -rf rust-bridge/target
