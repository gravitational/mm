VERSION ?= $(shell git describe --long --tags --always|awk -F'[.-]' '{print $$1 "." $$2 "." $$4}')

# Build
APPLICATION ?= $(shell basename $(CURDIR))
BUILD_DIR ?= bin

.PHONY: all
all: clean $(BUILD_DIR)

$(BUILD_DIR):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o $(BUILD_DIR)/$(APPLICATION) .

.PHONY: install
install:
	go install .

.PHONY: clean
clean:
	-rm -r $(BUILD_DIR)

# Docker
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/$(APPLICATION)
DOCKERFLAGS := --rm=true $(NOROOT) -v $(CURDIR):$(SRCDIR) -w $(SRCDIR)
BUILDIMAGE := quay.io/gravitational/debian-venti:go1.7-jessie

.PHONY: what-version
what-version:
	@echo $(VERSION)

.PHONY: docker-build
docker-build:
	docker run $(DOCKERFLAGS) $(BUILDIMAGE) make

.PHONY: docker-image
docker-image:
	docker build --rm --pull --tag $(APPLICATION):$(VERSION) .
	docker tag $(APPLICATION):$(VERSION) $(APPLICATION):latest

.PHONY: docker-clean
docker-clean:
	-docker rmi -f $(docker images $(APPLICATION) -q)

# Dev
.PHONY: format
format:
	goimports -w .

GOMETALINTER_REQUIRED_FLAGS := --vendor --tests --errors
# gotype is broken, see https://github.com/alecthomas/gometalinter/issues/91
GOMETALINTER_COMMON_FLAGS := --concurrency 3 --deadline 60s --line-length 120 --enable lll --disable gotype

.PHONY: lint
lint:
	gometalinter \
		$(GOMETALINTER_COMMON_FLAGS) \
		$(GOMETALINTER_REQUIRED_FLAGS) \
		./...

.PHONY: check
check:
	gometalinter \
		--enable goimports \
		--disable errcheck \
		--disable golint \
		--fast \
		$(GOMETALINTER_COMMON_FLAGS) \
		$(GOMETALINTER_REQUIRED_FLAGS) \
		./...

PROJECT_PKGS := $$(glide novendor)

.PHONY: test
test:
	for pkg in $(PROJECT_PKGS); do \
		go test -cover -v -race $$pkg || exit 1 ;\
	done

.PHONY: sloccount
sloccount:
	find . -path ./vendor -prune -o -name "*.go" -print0 | xargs -0 wc -l

.PHONY: info
info:
	depscheck -totalonly -tests $$(PROJECT_PKGS)

.PHONY: std-info
std-info:
	depscheck -stdlib -v $(PROJECT_PKGS)

PACKAGES := \
	golang.org/x/tools/cmd/goimports \
	github.com/Masterminds/glide \
	github.com/alecthomas/gometalinter \
	github.com/divan/depscheck

.PHONY: install-tools
install-tools:
	$(foreach pkg,$(PACKAGES),go get -u $(pkg);)
	gometalinter --install --update
