# Copyright 2024 Hewlett Packard Enterprise Development LP
.DEFAULT_GOAL := terraschema

.PHONY: test
test: 
	go test ./...

.PHONY: lint
lint: 
	golangci-lint run

.PHONY: all
all: test lint

.PHONY: terraschema
terraschema: 
	@go build .

.PHONY: clean
clean:
	@rm -f terraschema