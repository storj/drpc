.DEFAULT_GOAL = all

.PHONY: all
all: tidy docs generate lint vet test

.PHONY: check
check: generate vet

.PHONY: tidy
tidy:
	./scripts/run.sh '*' go mod tidy

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
	./scripts/run.sh -v 'examples' check-atomic-align ./...
	./scripts/run.sh -v 'examples' staticcheck ./...
	./scripts/run.sh -v 'examples' golangci-lint run

.PHONY: vet
vet:
	./scripts/run.sh '*' go vet ./...
	GOOS=linux   GOARCH=386   ./scripts/run.sh '*' go vet ./...
	GOOS=linux   GOARCH=amd64 ./scripts/run.sh '*' go vet ./...
	GOOS=linux   GOARCH=arm   ./scripts/run.sh '*' go vet ./...
	GOOS=linux   GOARCH=arm64 ./scripts/run.sh '*' go vet ./...
	GOOS=windows GOARCH=386   ./scripts/run.sh '*' go vet ./...
	GOOS=windows GOARCH=amd64 ./scripts/run.sh '*' go vet ./...
	GOOS=windows GOARCH=arm64 ./scripts/run.sh '*' go vet ./...
	GOOS=darwin  GOARCH=amd64 ./scripts/run.sh '*' go vet ./...
	GOOS=darwin  GOARCH=arm64 ./scripts/run.sh '*' go vet ./...

.PHONY: test
test:
	./scripts/run.sh '*'           go test ./...              -race -count=1 -bench=. -benchtime=1x
	./scripts/run.sh 'integration' go test ./... -tags=gogo   -race -count=1 -bench=. -benchtime=1x
	./scripts/run.sh 'integration' go test ./... -tags=custom -race -count=1 -bench=. -benchtime=1x
