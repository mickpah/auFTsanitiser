# justfile for auFTsanitiser - plain-text report sanitiser for AU HL7 v2.4 messaging.
# Run `just` or `just --list` to see available recipes.

# Binary and module settings
binary := "auFTsanitiser"
pkg := "."

# Show the list of recipes (default when running bare `just`).
default:
    @just --list

# --- Build ---

# Build the local (macOS-native) binary. This is the default for local dev and testing.
build:
    go build -o {{binary}} {{pkg}}

# Install the binary into GOBIN / GOPATH/bin.
install:
    go install {{pkg}}

# Cross-compile self-contained static binaries for all targets into dist/.
# macOS is the local dev/testing default; linux/windows are deployment targets.
dist:
    CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/{{binary}}-darwin-arm64    {{pkg}}
    CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/{{binary}}-darwin-amd64    {{pkg}}
    CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/{{binary}}-linux-amd64     {{pkg}}
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/{{binary}}-windows-amd64.exe {{pkg}}
    @echo "built darwin (arm64+amd64), linux-amd64, windows-amd64 in dist/"

# Build a Debian (amd64) .deb package into dist/ (for Debian 13 / trixie).
deb:
    ./scripts/build-deb.sh

# --- Quality ---

# Format all Go source in place.
fmt:
    gofmt -w .

# Fail if any Go source is not gofmt-clean (CI-friendly).
fmt-check:
    @test -z "$(gofmt -l .)" || { echo "gofmt needed on:"; gofmt -l .; exit 1; }

# Run go vet across all packages.
vet:
    go vet ./...

# Static checks: formatting + vet.
lint: fmt-check vet

# Guard: no PHI-shaped identifiers or names in the tracked HL7/text fixtures.
# Pass extra paths to check a corpus before committing it (see scripts/anonymise-hl7.py).
phi-scan *paths:
    python3 scripts/scan-for-phi.py --self-test
    python3 scripts/anonymise-hl7.py --self-test
    python3 scripts/scan-for-phi.py {{paths}}

# --- Tests ---

# Run the full test suite.
test:
    go test ./...

# Rhino unit tests for the Mirth code-template logic (needs java on PATH). The channel
# lives in the gitignored private/deployment tree, so this is a local-only recipe.
rhino:
    RHINO_JAR=private/deployment/tests/rhino/rhino-1.7.14.jar bash private/deployment/verify/run_rhino_tests.sh private/deployment/tests/rhino private/deployment/channel/code-templates

# Run tests verbosely, no cache.
test-v:
    go test -count=1 -v ./...

# Run tests with the race detector.
race:
    go test -race ./...

# Produce a coverage profile and print the summary.
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -n 1

# Open the HTML coverage report.
cover-html: cover
    go tool cover -html=coverage.out

# --- Run ---

# Sanitise a text file to stdout, e.g. `just run testdata/report.txt`.
run file:
    go run {{pkg}} {{file}}

# Print the tool version.
version:
    go run {{pkg}} --version

# Extended fuzz run over the release properties (charset invariant,
# determinism, idempotence, delimiter safety, escape round-trip).
fuzz time="60s":
    go test -run FuzzSanitise -fuzz FuzzSanitise -fuzztime {{time}} ./internal/sanitise/
    go test -run FuzzEscapeRoundTrip -fuzz FuzzEscapeRoundTrip -fuzztime {{time}} ./internal/hl7escape/

# Benchmarks incl. the NFR-004 1 MB budget assertion.
bench:
    go test -bench . -benchmem -run TestPerformanceBudget -v ./internal/sanitise/

# Cross-platform parity (SC-004): run the golden corpus against the native
# dist binary and byte-compare; CI runs this same recipe on each OS in the
# matrix so identical fixtures must produce identical bytes everywhere.
parity: dist
    #!/usr/bin/env bash
    set -euo pipefail
    bin="dist/{{binary}}-$(go env GOOS)-$(go env GOARCH)"
    [ "$(go env GOOS)" = "windows" ] && bin="$bin.exe"
    fail=0
    for in in testdata/*.input.txt; do
        golden="${in%.input.txt}.golden.txt"
        if ! "$bin" "$in" | cmp -s - "$golden"; then
            echo "parity mismatch: $in"; fail=1
        fi
    done
    [ $fail -eq 0 ] && echo "parity OK: $(ls testdata/*.input.txt | wc -l | tr -d ' ') fixtures byte-identical"

# --- Maintenance ---

# Tidy module dependencies.
tidy:
    go mod tidy

# Remove build artifacts.
clean:
    rm -f {{binary}} {{binary}}.exe coverage.out
    rm -rf dist

# Full local gate: lint, PHI scan, Go tests (what CI runs). Run `just rhino` as well when private/ is present.
check: lint phi-scan test

# Build everything releasable: clean, check, dist.
all: clean check dist
