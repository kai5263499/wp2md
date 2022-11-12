which = $(shell which $1 2> /dev/null || echo $1)

GO_PATH := $(call which,go)
$(GO_PATH):
	$(error Missing go)

TEST_ARGS := -cover -v -count=1
test: 
	@$(GO_PATH) test $(TEST_ARGS) ./... -tags=!intg
.PHONY: test

LINTER_PATH := $(call which,golangci-lint)
$(LINTER_PATH):
	$(error Missing golangci: https://golangci-lint.run/usage/install)
lint:
	@rm -rf ./vendor
	@$(GO_PATH) mod vendor
	export GOMODCACHE=./vendor
	golangci-lint run
.PHONY: test