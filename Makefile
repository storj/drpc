.DEFAULT_GOAL = all

.PHONY: all
all: tidy docs generate lint test

.PHONY: quick
quick: generate test

.PHONY: docs
docs:
	./scripts/docs.sh

.PHONY: generate
generate:
	./scripts/run.sh '*' go generate ./...

.PHONY: lint
lint:
	./scripts/run.sh -v 'examples' check-copyright
	./scripts/run.sh -v 'examples' check-large-files
	./scripts/run.sh -v 'examples' check-imports ./...
	./scripts/run.sh -v 'examples' check-peer-constraints
	./scripts/run.sh -v 'examples' check-atomic-align ./...
	./scripts/run.sh -v 'examples' check-errs ./...
	./scripts/run.sh -v 'examples' staticcheck ./...
	./scripts/run.sh -v 'examples' golangci-lint -j=2 run

.PHONY: test
test:
	./scripts/run.sh '*'           go test ./...              -race -count=1 -bench=. -benchtime=1x
	./scripts/run.sh 'integration' go test ./... -tags=gogo   -race -count=1 -bench=. -benchtime=1x
	./scripts/run.sh 'integration' go test ./... -tags=custom -race -count=1 -bench=. -benchtime=1x

.PHONY: tidy
tidy:
	./scripts/run.sh '*' go mod tidy
