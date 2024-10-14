# Build variables
VERSION ?= $(shell git describe --tags --always)

# Go variables
GO      ?= go
GOOS    ?= $(shell $(GO) env GOOS)
GOARCH  ?= $(shell $(GO) env GOARCH)
GOHOST  ?= GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO)

.PHONY: all
all: help

###############
##@ Development

.PHONY: test
test:   ## Run tests
	@ $(MAKE) --no-print-directory log-$@
	$(GOHOST) test -timeout 60s -race -covermode atomic -coverprofile coverage.out -v ./...

.PHONY: lint
lint: ## Run linters
	@ $(MAKE) --no-print-directory log-$@
	golangci-lint run

########
##@ Help

.PHONY: help
help:   ## Display this help
	@awk \
		-v "col=\033[36m" -v "nocol=\033[0m" \
		' \
			BEGIN { \
				FS = ":.*##" ; \
				printf "Usage:\n  make %s<target>%s\n", col, nocol \
			} \
			/^[a-zA-Z_-]+:.*?##/ { \
				printf "  %s%-12s%s %s\n", col, $$1, nocol, $$2 \
			} \
			/^##@/ { \
				printf "\n%s%s%s\n", nocol, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk \
			'BEGIN { \
				FS = ":.*?## " \
			}; \
			{ \
				printf "\033[36m==> %s\033[0m\n", $$2 \
			}'
