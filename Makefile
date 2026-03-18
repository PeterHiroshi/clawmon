.PHONY: build build-daemon build-tui test test-daemon test-tui test-e2e lint run-daemon run-tui clean

# Build
build: build-daemon build-tui

build-daemon:
	cd daemon && cargo build --release

build-tui:
	cd tui && go build -o clawmon-tui ./cmd/clawmon-tui

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
	rm -f tui/clawmon-tui
