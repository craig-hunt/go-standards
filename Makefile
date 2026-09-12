GO ?= go

GOLANGCI_LINT_VERSION := v2.13.2
GREMLINS_VERSION := v0.6.0
GOVULNCHECK_VERSION := v1.8.0
GITLEAKS_VERSION := v8.30.1
POSTGRES_IMAGE ?= postgres:17-alpine
INTEGRATION_TAG := integration

export POSTGRES_IMAGE

GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GREMLINS := $(GO) run github.com/go-gremlins/gremlins/cmd/gremlins@$(GREMLINS_VERSION)
GOVULNCHECK := $(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
GITLEAKS := $(GO) run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION)
WITH_TEST_POSTGRES := ./scripts/with-test-postgres.sh

.PHONY: verify format-check vet lint test integration mutation vulncheck secrets

verify: format-check vet lint test integration mutation vulncheck secrets

format-check:
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "$$unformatted"; exit 1; fi

vet:
	$(GO) vet ./...
	$(GO) vet -tags $(INTEGRATION_TAG) ./...

lint:
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test -race -count=1 ./...

integration:
	$(WITH_TEST_POSTGRES) $(GO) test -race -count=1 -tags $(INTEGRATION_TAG) ./...

mutation:
	$(WITH_TEST_POSTGRES) $(GREMLINS) unleash

vulncheck:
	$(GOVULNCHECK) ./...

secrets:
	$(GITLEAKS) dir --redact --verbose .
