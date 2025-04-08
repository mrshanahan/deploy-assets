.DEFAULT_GOAL: build

INSTALL_DIR ?= ~/bin

build:
	go build -o "$(CURDIR)/cmd/deploy-assets" "$(CURDIR)/cmd/deploy-assets.go"

test:
	go test "$(CURDIR)/cmd" "$(CURDIR)/internal/*"

run:
	go run "$(CURDIR)/cmd/deploy-assets.go"

install:
	go build -o $(INSTALL_DIR)/deploy-assets "$(CURDIR)/cmd/deploy-assets.go"

.PHONY: build test run install