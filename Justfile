setup: tidy build

[group('check')]
[group('fmt')]
fmt:
    gofmt -w $(find . -name '*.go')
[group('check')]
[group('fmt')]
fmt-check:
    @diff=$(gofmt -d $(find . -name '*.go')); \
    if [ -n "$diff" ]; then \
        echo "$diff"; \
        exit 1; \
    fi
[group('check')]
vet:
    go vet ./...
[group('check')]
test:
    go test ./...
[group('check')]
vuln:
    go tool govulncheck ./...
[group('check')]
tidy:
    go mod tidy
[group('check')]
build:
    go build -o ./bin/rellog ./cmd/rellog

[group('check')]
license-check:
    go tool go-licenses check --include_tests ./...
[group('release')]
license-save:
    go tool go-licenses save ./cmd/rellog --save_path third_party_licenses

[group('check')]
lint:
    golangci-lint run

[group('check')]
ci: coverage fmt-check vet vuln license-check
[group('check')]
check: lint ci

[group('check')]
coverage:
    @bash scripts/test/coverage.sh

[group('release')]
release-tag version:
    rellog ready {{ version }}

    git tag -a {{ version }} -m "rellog {{ version }}"

latest_tag := `git tag | grep -v latest | sort -V | tail -n 1`
[group('release')]
release-latest:
    rellog ready {{ latest_tag }}

    git tag -d latest 2>/dev/null || true
    git tag -a latest -m "rellog latest({{ latest_tag }})"

[group('release')]
release-run:
    git push origin --delete latest
    git push --tags

[private]
release-tag-del version:
    # this command is for testint and debuggint. this would be removed in future.
    git tag -d latest 2>/dev/null || true
    git push origin --delete latest 2>/dev/null || true

    git tag -d {{ version }} 2>/dev/null || true
    git push origin --delete {{ version }} 2>/dev/null || true
