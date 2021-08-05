PKG=github.com/cyverse/purgeman
GO111MODULE=on
GOPROXY=direct
GOPATH=$(shell go env GOPATH)

.EXPORT_ALL_VARIABLES:

.PHONY: build
build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -o bin/purgeman ./cmd/

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
