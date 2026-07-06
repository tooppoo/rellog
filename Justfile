setup: tidy build

fmt:
    gofmt -w $(find . -name '*.go')

fmt-check:
    @diff=$(gofmt -d $(find . -name '*.go')); \
    if [ -n "$diff" ]; then \
        echo "$diff"; \
        exit 1; \
    fi

vet:
    go vet ./...

test:
    go test ./...

vuln:
    go tool govulncheck ./...

tidy:
    go mod tidy

build:
    go build -o ./bin/rellog ./cmd/rellog

license-check:
    go tool go-licenses check --include_tests ./...

license-save:
    go tool go-licenses save ./cmd/rellog --save_path third_party_licenses

lint:
    golangci-lint run

ci: coverage fmt-check vet vuln license-check

check: lint ci

coverage:
    @bash scripts/test/coverage.sh

release-tag version:
    rellog ready {{ version }}
    git tag -d latest 2>/dev/null || true
    git push origin --delete latest 2>/dev/null || true

    git tag -a latest -m "@latest({{ version}})"
    git tag -a {{ version }} -m "@{{ version }}"

[private]
release-tag-del version:
    # this command is for testint and debuggint. this would be removed in future.
    git tag -d latest 2>/dev/null || true
    git push origin --delete latest 2>/dev/null || true

    git tag -d {{ version }} 2>/dev/null || true
    git push origin --delete {{ version }} 2>/dev/null || true
