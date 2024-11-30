
.PHONY: build
build:
	go build -o bsky-cli cmd/cli/main.go
	go build -o list-manager cmd/list-manager/main.go


all: build
