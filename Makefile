.DEFAULT_GOAL = all

.PHONY: all
all: download docs generate build lint test tidy

.PHONY: docs
docs:
	./scripts/docs.sh

.PHONY: generate
generate:
	./scripts/run.sh go generate ./...

.PHONY: build
build:
	./scripts/run.sh go build ./...

.PHONY: lint
lint:
	./scripts/run.sh check-copyright
	./scripts/run.sh check-large-files
	./scripts/run.sh check-imports ./...
	./scripts/run.sh check-peer-constraints
	./scripts/run.sh check-atomic-align ./...
	./scripts/run.sh check-monkit ./...
	./scripts/run.sh check-errs ./...
	./scripts/run.sh staticcheck ./...
	./scripts/run.sh golangci-lint -j=2 run

.PHONY: test
test:
	./scripts/run.sh go test ./... -race -count=1 -bench=. -benchtime=1x

.PHONY: download
download:
	./scripts/run.sh go mod download

.PHONY: tidy
tidy:
	./scripts/run.sh go mod tidy

.PHONY: tools
tools:
	./scripts/install-tools.sh
