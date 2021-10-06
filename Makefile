PKG=github.com/cyverse/purgeman
VERSION=v0.3.0
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X '${PKG}/pkg/commons.serviceVersion=${VERSION}' -X '${PKG}/pkg/commons.gitCommit=${GIT_COMMIT}' -X '${PKG}/pkg/commons.buildDate=${BUILD_DATE}'"
GO111MODULE=on
GOPROXY=direct
GOPATH=$(shell go env GOPATH)

.EXPORT_ALL_VARIABLES:

.PHONY: build
build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags=${LDFLAGS} -o bin/purgeman ./cmd/

.PHONY: release
release: build
	mkdir -p release
	mkdir -p release/bin
	cp bin/purgeman release/bin
	mkdir -p release/install
	cp install/purgeman.conf release/install
	cp install/purgeman.service release/install
	cp install/README.md release/install
	cp Makefile.release release/Makefile
	cd release && tar zcvf ../purgeman.tar.gz *

.PHONY: install_centos
install_centos:
	cp bin/purgeman /usr/bin
	cp install/purgeman.service /usr/lib/systemd/system/
	id -u purgeman &> /dev/null || adduser -r -d /dev/null -s /sbin/nologin purgeman
	mkdir -p /etc/purgeman
	cp install/purgeman.conf /etc/purgeman
	chown purgeman /etc/purgeman/purgeman.conf
	chmod 660 /etc/purgeman/purgeman.conf

.PHONY: install_ubuntu
install_ubuntu:
	cp bin/purgeman /usr/bin
	cp install/purgeman.service /etc/systemd/system/
	id -u purgeman &> /dev/null || adduser --system --home /dev/null --shell /sbin/nologin purgeman
	mkdir -p /etc/purgeman
	cp install/purgeman.conf /etc/purgeman
	chown purgeman /etc/purgeman/purgeman.conf
	chmod 660 /etc/purgeman/purgeman.conf

.PHONY: uninstall
uninstall:
	rm -f /usr/bin/purgeman
	rm -f /etc/systemd/system/purgeman.service
	rm -f /usr/lib/systemd/system/purgeman.service
	userdel purgeman | true
	rm -rf /etc/purgeman
