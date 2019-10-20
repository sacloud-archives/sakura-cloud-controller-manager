VETARGS?=-all
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
BIN_NAME?=sakura-cloud-controller-manager
CURRENT_VERSION = $(gobump show -r version/)
GO_FILES?=$(shell find . -name '*.go')

BUILD_LDFLAGS = "-s -w \
	  -X github.com/sacloud/sakura-cloud-controller-manager/version.Revision=`git rev-parse --short HEAD`"

.PHONY: default
default: test build

.PHONY: run
run:
	go run $(CURDIR)/*.go $(ARGS)

.PHONY: clean
clean:
	rm -Rf bin/*

.PHONY: tools
tools:
	GO111MODULE=off go get -u golang.org/x/tools/cmd/goimports
	GO111MODULE=off go get -u github.com/motemen/gobump/cmd/gobump
	GO111MODULE=off go get -u golang.org/x/lint/golint

.PHONY: build
build: bin/sakura-cloud-controller-manager

bin/sakura-cloud-controller-manager: $(GO_FILES)
	GOOS=`go env GOOS` GOARCH=`go env GOARCH` CGO_ENABLED=0 \
                go build \
                    -ldflags $(BUILD_LDFLAGS) \
                    -o bin/sakura-cloud-controller-manager \
                    *.go

.PHONY: test
test: lint
	go test ./... $(TESTARGS) -v -timeout=30m -parallel=4 ;

.PHONY: lint
lint: fmt
	go list ./... | xargs -L1 golint

.PHONY: fmt
fmt:
	gofmt -s -l -w $(GOFMT_FILES)

goimports:
	goimports -w $(GOFMT_FILES)

.PHONY: version bump-patch bump-minor bump-major
version:
	gobump show -r version/

bump-patch:
	gobump patch -w version/

bump-minor:
	gobump minor -w version/

bump-major:
	gobump major -w version/
