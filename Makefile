.DEFAULT_GOAL := help

export GO111MODULE := on
export PATH := $(CURDIR)/.go-packages/bin:$(PATH)

# This is a magic code to output help message at default
# see https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY:help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY:fmt
fmt: ## Run `go fmt`
	go fmt $$(go list ./... | grep -v vendor)

.PHONY:test
test: ## Run all tests
	go test -cover $$(go list ./... | grep -v vendor)

.PHONY:testv
testv: ## Run all tests with verbose outputing.
	go test -v -cover $$(go list ./... | grep -v vendor)

.PHONY:testcov
testcov:
	gocov test $$(go list ./... | grep -v vendor) | gocov-html > coverage-report.html

.PHONY:dev
dev: ## Build dev binary
	@bash -c $(CURDIR)/build/scripts/dev.sh

.PHONY:dist
dist: bindata ## Build dist binaries
	@bash -c $(CURDIR)/build/scripts/dist.sh

.PHONY:clean
clean: ## Clean build outputs
	@bash -c $(CURDIR)/build/scripts/clean.sh

.PHONY:packaging
packaging: ## Create packages (now support RPM only)
	@bash -c $(CURDIR)/build/scripts/packaging.sh

.PHONY: installtools
installtools: ## Install dev tools
	GOPATH=$(CURDIR)/.go-packages && \
      go get github.com/mitchellh/gox && \
      go get github.com/axw/gocov/gocov && \
      go get github.com/matm/gocov-html && \
      go clean -modcache

.PHONY: cleantools
cleantools:
	GOPATH=$(CURDIR)/.go-packages && go clean -modcache && rm -rf $(CURDIR)/.go-packages

.PHONY:deps
deps: ## Install dependences.
	go mod tidy


