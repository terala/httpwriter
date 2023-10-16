VENDOR_DIR = $(dir Makefile)vendor
OUTDIR = $(dir Makefile)out
GO_FILES := $(WILDCARD *.go)


.PHONY: outdir
$(OUTDIR):
	mkdir -p $(OUTDIR)
outdir: $(OUTDIR)

.PHONY: vendor
$(VENDOR_DIR)/modules.txt: go.mod
	go mod vendor
vendor: $(VENDOR_DIR)/modules.txt

.PHONY: lint
lint:
	~/go/bin/golangci-lint run

.PHONY: tidy
tidy: go.mod
	go mod tidy -v

.PHONY: test
test: $(GO_FILES) outdir
	go test -v -coverprofile=$(OUTDIR)/coverage.out ./...

.PHONY: coverage_file
$(OUTDIR)/coverage.out: $(GO_FILES) test
coverage_file: $(OUTDIR)/coverage.out

.PHONY: coverage
coverage: coverage_file
	go tool cover -html=$(OUTDIR)/coverage.out -o $(OUTDIR)/coverage.html

build: lint test coverage

clean:
	rm -rf "$(OUTDIR)"

all: test

.DEFAULT_GOAL := build
