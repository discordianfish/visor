VERSION  := $(shell cat VERSION)
GOPATH   ?= $(PWD)
GOBIN    ?= $(GOPATH)/bin
LDFLAGS  := -ldflags "-X main.VERSION $(VERSION)"
GOFLAGS  := -x $(LDFLAGS)
PKGPATH  := $(GOPATH)/src/github.com/soundcloud/visor
GOARCH   ?= amd64

# LOCAL #

default:
	@go build $(LDFLAGS) ./cmd/visor
	@echo built ./visor v$(VERSION)

install: $(PKGPATH)
	GOPATH=$(GOPATH) go get $(GOFLAGS) -d ./cmd/visor
	GOPATH=$(GOPATH) GOBIN=$(GOBIN) go install $(GOFLAGS) ./cmd/visor
	@echo "built $(GOBIN)/visor v$(VERSION)"

$(PKGPATH):
	mkdir -p $(shell dirname $(PKGPATH))
	ln -sf $(PWD) $(PKGPATH)

test:
	go test

# DIST #

dist: linux darwin

linux darwin:
	GOOS=$@ CGO_ENABLED=0 GOARCH=$(GOARCH) go build $(LDFLAGS) -o bin/$@/visor ./cmd/visor

# DEBIAN PACKAGING #

DEB_NAME=visor
DEB_URL=http://github.com/soundcloud/visor
DEB_VERSION=$(VERSION)
DEB_DESCRIPTION=A command line interface for visor
DEB_MAINTAINER=Daniel Bornkessel <daniel@soundcloud.com>

include deb.mk

debroot:
	GOBIN=$(DEB_ROOT)/usr/bin $(MAKE)

# BUILD #

build: clean debroot debbuild

clean: debclean
	GOPATH=$(GOPATH) go clean
	rm -rf bin src pkg

.PHONY: test
