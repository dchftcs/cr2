BINARY := cr
CMD := ./cmd/cr
BUILD_DIR := bin
BUILD_OUTPUT := $(BUILD_DIR)/$(BINARY)

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=
INSTALL_PATH := $(DESTDIR)$(BINDIR)/$(BINARY)

.PHONY: all build install test clean

all: build

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_OUTPUT) $(CMD)

install: build
	mkdir -p $(DESTDIR)$(BINDIR)
	cp $(BUILD_OUTPUT) $(INSTALL_PATH)
	chmod 0755 $(INSTALL_PATH)

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
