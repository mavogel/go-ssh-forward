# Build variables
VERSION ?= $(shell git describe --tags --always)

# Go variables
GO      ?= go
GOOS    ?= $(shell $(GO) env GOOS)
GOARCH  ?= $(shell $(GO) env GOARCH)
GOHOST  ?= GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO)

# Tool versions
GOLANGCI_VERSION  = 1.39.0
GITCHGLOG_VERSION = 0.14.2

.PHONY: all
all: help

###############
##@ Development

.PHONY: test
test:   ## Run tests
	@ $(MAKE) --no-print-directory log-$@
	$(GOHOST) test -race -covermode atomic -coverprofile cover.out -v ./...

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | BINARY=golangci-lint bash -s -- v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

.PHONY: lint
lint: bin/golangci-lint ## Run linters
	@ $(MAKE) --no-print-directory log-$@
	bin/golangci-lint run

###########
##@ Release

bin/git-chglog: bin/git-chglog-${GITCHGLOG_VERSION}
	@ln -sf git-chglog-${GITCHGLOG_VERSION} bin/git-chglog
bin/git-chglog-${GITCHGLOG_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/git-chglog/git-chglog/releases/download/v${GITCHGLOG_VERSION}/git-chglog_${GITCHGLOG_VERSION}_${OS}_amd64.tar.gz | tar -zOxf - git-chglog > ./bin/git-chglog-${GITCHGLOG_VERSION} && chmod +x ./bin/git-chglog-${GITCHGLOG_VERSION}

.PHONY: changelog
changelog: bin/git-chglog   ## Generate changelog
	@ $(MAKE) --no-print-directory log-$@
	bin/git-chglog --next-tag $(VERSION) -o CHANGELOG.md

.PHONY: release
release: changelog   ## Release a new tag
	@ $(MAKE) --no-print-directory log-$@
	git add CHANGELOG.md
	git commit -m "chore: update changelog for $(VERSION)"
	git tag $(VERSION)
	git push origin master $(VERSION)

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
