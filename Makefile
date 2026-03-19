.PHONY: build build-daemon build-tui test test-daemon test-tui test-e2e lint run-daemon run-tui clean build-linux build-macos

# Build
build: build-daemon build-tui

build-daemon:
	cd daemon && cargo build --release

build-tui:
	cd tui && go build -o clawmon-tui ./cmd/clawmon-tui

# Cross-platform builds
build-linux: build-linux-daemon build-linux-tui

build-linux-daemon:
	cd daemon && cargo build --release --target x86_64-unknown-linux-gnu

build-linux-tui:
	cd tui && GOOS=linux GOARCH=amd64 go build -o clawmon-tui-linux ./cmd/clawmon-tui

build-macos: build-macos-daemon build-macos-tui

build-macos-daemon:
	cd daemon && cargo build --release --target x86_64-apple-darwin

build-macos-tui:
	cd tui && GOOS=darwin GOARCH=amd64 go build -o clawmon-tui-macos ./cmd/clawmon-tui

# Test
test: test-daemon test-tui

test-daemon:
	cd daemon && cargo test

test-tui:
	cd tui && go test ./...

test-e2e: test-e2e-daemon test-e2e-tui

test-e2e-daemon:
	cd daemon && cargo test --test 'e2e_*'

test-e2e-tui:
	cd tui && go test ./tests/e2e/...

# Lint
lint: lint-daemon lint-tui

lint-daemon:
	cd daemon && cargo clippy -- -W warnings

lint-tui:
	cd tui && go vet ./...

# Run
run-daemon:
	cd daemon && cargo run --release

run-tui:
	cd tui && go run ./cmd/clawmon-tui

# Clean
clean:
	cd daemon && cargo clean
	rm -f tui/clawmon-tui tui/clawmon-tui-linux tui/clawmon-tui-macos
