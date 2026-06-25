
.PHONY: setup
setup: tidy build

.PHONY: fmt
fmt:
	gofmt -w $$(find . -name '*.go')

.PHONY: fmt-check
fmt-check:
	@diff=$$(gofmt -d $$(find . -name '*.go')); \
	if [ -n "$$diff" ]; then \
		echo "$$diff"; \
		exit 1; \
	fi

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: vuln
vuln:
	go tool govulncheck ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: build
build:
	go build -o ./bin/rellog ./cmd/rellog

.PHONY: license-check
license-check:
	go tool go-licenses check --include_tests ./...

.PHONY: license-save
license-save:
	go tool go-licenses save ./cmd/git-kura --save_path third_party_licenses

.PHONY: lint
lint:
	golangci-lint run

.PHONY: ci
ci: fmt-check vet coverage vuln license-check

.PHONY: check
check: lint ci

.PHONY: coverage
coverage:
	@bash scripts/test/coverage.sh
