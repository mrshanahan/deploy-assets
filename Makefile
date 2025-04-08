.DEFAULT_GOAL: build

build:
	go build -o $(CURDIR)/cmd/deploy-assets $(CURDIR)/cmd/deploy-assets.go

test:
	go test $(CURDIR)/cmd $(CURDIR)/internal/*

run:
	go run $(CURDIR)/cmd/deploy-assets.go

install:
	go build -o ~/bin/deploy-assets $(CURDIR)/cmd/deploy-assets.go