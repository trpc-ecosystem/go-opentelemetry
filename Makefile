ALL_SRC := $(shell find . -name '*.go' \
							-not -path './opentelemetry/proto/*' \
							-type f | sort)
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

MISSPELL=misspell -error
IMPI=impi

GOTEST_MIN = go test -gcflags=all=-l -timeout 30s -vet off
GOTEST = $(GOTEST_MIN) -race
GOVET = go vet -printfuncs=addDyeAndCallFunc
GOTEST_WITH_COVERAGE = $(GOTEST) -coverprofile=coverage.out -covermode=atomic -coverpkg=./...

.PHONY: precommit
precommit: fmt vet lint build test examples

.PHONY: impi
impi:
	@$(IMPI) --local trpc-system/go-opentelemetry --scheme stdThirdPartyLocal --skip example/trpc/protocol --skip opentelemetry/proto ./...

.PHONY: misspell
misspell:
	$(MISSPELL) $(ALL_DOC)

.PHONY: lint
lint:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go mod tidy in $${dir}"; \
	  (cd "$${dir}" && \
	    go mod tidy); \
	done
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
    	  echo "golangci-lint in $${dir}"; \
    	  (cd "$${dir}" && \
    	    golangci-lint run --fix && \
    	    golangci-lint run); \
    done

.PHONY: test
test:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go test ./... + race in $${dir}"; \
	  (cd "$${dir}" && \
	    $(GOTEST) ./...); \
	done

.PHONY: vet
vet:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go vet ./... in $${dir}"; \
	  (cd "$${dir}" && \
	    $(GOVET) ./...); \
	done

.PHONY: mod-tidy
mod-tidy:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
  		echo "go mod tidy ./.. in $${dir}"; \
  		(cd "$${dir}" && go mod tidy); \
  	done

.PHONY: build
build:
	@set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  (cd "$${dir}" && \
	    go build -v ./...; \
		echo "Build $${dir} success!"; \
	); \
	done
.PHONY: examples
examples:
	@cd example/basic; \
	go build -v
	@echo "Build basic example success!"

	@cd example/log; \
	go build -v
	@echo "Build log example success!"

.PHONY: fmt
fmt:
	gofmt -w -s .
	goimports -w -local trpc-system/go-opentelemetry ./

