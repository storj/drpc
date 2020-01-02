.DEFAULT_GOAL = check

.PHONY: check
check: build test lint

.PHONY: build
build:
	./scripts/build.sh

.PHONY: docs
docs:
	./scripts/docs.sh

.PHONY: download
download:
	./scripts/download.sh

.PHONY: lint
lint:
	./scripts/lint.sh

.PHONY: tidy
tidy:
	./scripts/tidy.sh

.PHONY: test
test:
	./scripts/test.sh
